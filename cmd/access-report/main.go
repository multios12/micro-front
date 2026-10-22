package main

import (
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

// アクセスレポートの集計・表示設定。変更する場合はここを編集する。
const (
	reportTimezone = "Asia/Tokyo" // 日別集計のタイムゾーン
	scanMin404     = 100          // スキャン疑いとする1日あたりの最小404件数
	scanMinPaths   = 20           // スキャン疑いとする1日あたりの最小パス種類数
)

//go:embed templates/access-report-crawlers.json
var defaultCrawlerRules []byte

type inputs []string

func (s *inputs) String() string     { return strings.Join(*s, ",") }
func (s *inputs) Set(v string) error { *s = append(*s, v); return nil }

type options struct {
	Inputs                                      inputs
	LogDir, LogName, Host, State, Output, Rules string
	loc                                         *time.Location
	compiledRules                               []compiledCrawlerRule
	patternsText                                []string
	BlogDB                                      string
	blogDBExplicit                              bool
}
type crawlerRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	Enabled     *bool  `json:"enabled"`
}

type compiledCrawlerRule struct {
	ID, Name, Pattern string
	Regex             *regexp.Regexp
}
type ruleFile struct {
	Version int           `json:"version"`
	Rules   []crawlerRule `json:"rules"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// 明示した引数を優先し、省略した引数だけ環境変数で補う。
// INPUT は改行区切りで複数ファイルを指定できる。
func applyEnv(f *flag.FlagSet) error {
	explicit := map[string]bool{}
	f.Visit(func(v *flag.Flag) { explicit[v.Name] = true })
	var envErr error
	f.VisitAll(func(v *flag.Flag) {
		if explicit[v.Name] || envErr != nil {
			return
		}
		name := "ACCESS_REPORT_" + strings.ToUpper(strings.ReplaceAll(v.Name, "-", "_"))
		value, ok := os.LookupEnv(name)
		if !ok {
			return
		}
		values := []string{value}
		if v.Name == "input" {
			values = strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
		}
		for _, value := range values {
			if v.Name == "input" && value == "" {
				continue
			}
			if err := f.Set(v.Name, value); err != nil {
				envErr = fmt.Errorf("%s: %w", name, err)
				return
			}
		}
	})
	return envErr
}

func run(ctx context.Context, args []string, stderr io.Writer) error {
	o := options{}
	f := flag.NewFlagSet("access-report", flag.ContinueOnError)
	f.SetOutput(stderr)
	f.Var(&o.Inputs, "input", "入力ログ（複数指定可能）")
	f.StringVar(&o.LogDir, "log-dir", "", "ログディレクトリ")
	f.StringVar(&o.LogName, "log-name", "access.log.json", "現行ログ名")
	f.StringVar(&o.Host, "host", "", "対象ホスト（必須）")
	f.StringVar(&o.State, "state", "./data/access-report-state/state.db", "状態DB")
	f.StringVar(&o.Output, "output", "./data/access-report/index.html", "出力HTML")
	f.StringVar(&o.Rules, "crawler-rules", "./config/access-report-crawlers.json", "クローラールールJSON")
	f.StringVar(&o.BlogDB, "blog-db", "./data/app.db", "記事タイトル取得用DB（読み取り専用、空文字で無効）")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("位置引数は使用できません")
	}
	if err := applyEnv(f); err != nil {
		return err
	}
	f.Visit(func(f *flag.Flag) {
		if f.Name == "blog-db" {
			o.blogDBExplicit = true
		}
	})
	o.Host = normalizeHost(o.Host)
	if o.Host == "" {
		return errors.New("--host が必要です")
	}
	if o.LogDir != "" && len(o.Inputs) > 0 {
		return errors.New("--log-dir と --input は同時指定できません")
	}
	if o.LogDir == "" && len(o.Inputs) == 0 {
		return errors.New("--input または --log-dir が必要です")
	}
	var err error
	o.loc, err = time.LoadLocation(reportTimezone)
	if err != nil {
		return err
	}
	if err = validatePaths(o, nil); err != nil {
		return err
	}
	if err = o.loadRules(); err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(o.State), 0700); err != nil {
		return err
	}
	stateAbs, _ := filepath.Abs(o.State)
	outAbs, _ := filepath.Abs(o.Output)
	if stateAbs == outAbs {
		return errors.New("DBとHTMLの出力先は分けてください")
	}
	st, err := openState(ctx, o)
	if err != nil {
		return err
	}
	defer st.db.Close()
	st.log = stderr
	seen := map[int64]bool{}
	var processed int64
	for attempt := 0; attempt < 3; attempt++ {
		paths, scanErr := o.files()
		if scanErr != nil {
			if o.LogDir != "" && errors.Is(scanErr, os.ErrNotExist) && attempt < 2 {
				continue
			}
			return scanErr
		}
		if err = validatePaths(o, paths); err != nil {
			return err
		}
		var retryErr error
		for _, p := range paths {
			n, id, ingestErr := st.ingest(ctx, p, o)
			if ingestErr != nil {
				if o.LogDir != "" && (errors.Is(ingestErr, os.ErrNotExist) || errors.Is(ingestErr, io.EOF) || errors.Is(ingestErr, io.ErrUnexpectedEOF) || errors.Is(ingestErr, gzip.ErrHeader) || errors.Is(ingestErr, gzip.ErrChecksum)) {
					retryErr = ingestErr
					break
				}
				return fmt.Errorf("%s: %w", filepath.Base(p), ingestErr)
			}
			seen[id] = true
			processed += n
		}
		if retryErr == nil {
			break
		}
		if attempt == 2 {
			return fmt.Errorf("ログ再探索の上限に達しました: %w", retryErr)
		}
	}
	var valid int64
	if err = st.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(valid_lines),0) FROM log_generations").Scan(&valid); err != nil {
		return err
	}
	if valid == 0 {
		return errors.New("有効なアクセスログがありません")
	}
	rows, err := st.db.QueryContext(ctx, "SELECT id FROM log_generations WHERE sealed=0")
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if !seen[id] {
			fmt.Fprintln(stderr, "警告: 入力に未完了の旧ログ世代がありません:", id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if err = st.render(ctx, o); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "取り込み: %d行、出力: %s\n", processed, o.Output)
	return nil
}

func validatePaths(o options, paths []string) error {
	key := filepath.Join(filepath.Dir(o.State), filepath.Base(o.State)+".visitor.key")
	protected := []string{o.State, o.Output, o.Rules, key}
	if o.BlogDB != "" {
		protected = append(protected, o.BlogDB)
	}
	all := append(append([]string{}, paths...), protected...)
	for i, a := range all {
		for j := i + 1; j < len(all); j++ {
			// Multiple read-only inputs are rejected too, including hard links.
			b := all[j]
			aa, _ := filepath.Abs(a)
			bb, _ := filepath.Abs(b)
			ai, ae := os.Stat(a)
			bi, be := os.Stat(b)
			if aa == bb || (ae == nil && be == nil && os.SameFile(ai, bi)) {
				return errors.New("入力・出力・状態・設定ファイルのパスが重複しています")
			}
		}
	}
	return nil
}
func (o *options) loadRules() error {
	b, err := os.ReadFile(o.Rules)
	if errors.Is(err, os.ErrNotExist) {
		if err = createDefaultRules(o.Rules); err != nil {
			return err
		}
		b, err = os.ReadFile(o.Rules)
	}
	if err != nil {
		return err
	}
	var rf ruleFile
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err = d.Decode(&rf); err != nil {
		return err
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return errors.New("ルールJSON末尾が不正です")
	}
	if rf.Version != 1 || rf.Rules == nil {
		return errors.New("未対応ルール形式です")
	}
	ids := map[string]bool{}
	ps := map[string]bool{}
	for _, r := range rf.Rules {
		if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Name) == "" || ids[r.ID] || r.Pattern == "" || r.Enabled == nil {
			return errors.New("ルールの必須項目またはIDが不正です")
		}
		ids[r.ID] = true
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return err
		}
		if *r.Enabled {
			o.compiledRules = append(o.compiledRules, compiledCrawlerRule{r.ID, r.Name, r.Pattern, re})
			if !ps[r.Pattern] {
				o.patternsText = append(o.patternsText, r.Pattern)
				ps[r.Pattern] = true
			}
		}
	}
	sort.Strings(o.patternsText)
	return nil
}

func createDefaultRules(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = f.Write(defaultCrawlerRules); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}
func (o options) files() ([]string, error) {
	paths := []string(o.Inputs)
	if o.LogDir != "" {
		if filepath.Base(o.LogName) != o.LogName {
			return nil, errors.New("--log-name はファイル名のみ指定してください")
		}
		entries, err := os.ReadDir(o.LogDir)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(o.LogName)
		stem := strings.TrimSuffix(o.LogName, ext)
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() && (n == o.LogName || (strings.HasPrefix(n, stem+"-") && (strings.HasSuffix(n, ext) || strings.HasSuffix(n, ext+".gz")))) {
				paths = append(paths, filepath.Join(o.LogDir, n))
			}
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("入力ログが見つかりません: %w", os.ErrNotExist)
	}
	seen := map[string]bool{}
	for i, p := range paths {
		a, err := filepath.EvalSymlinks(p)
		if err != nil {
			return nil, err
		}
		a, err = filepath.Abs(a)
		if err != nil {
			return nil, err
		}
		if seen[a] {
			return nil, errors.New("入力ファイルが重複しています")
		}
		seen[a] = true
		paths[i] = a
	}
	sort.Slice(paths, func(i, j int) bool {
		ci, cj := filepath.Base(paths[i]) == o.LogName, filepath.Base(paths[j]) == o.LogName
		if ci != cj {
			return !ci
		}
		return paths[i] < paths[j]
	})
	return paths, nil
}
