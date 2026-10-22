package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixture struct{ dir, log, state, output, rules string }

func setup(t *testing.T) fixture {
	t.Helper()
	d := t.TempDir()
	f := fixture{dir: d, log: filepath.Join(d, "access.log.json"), state: filepath.Join(d, "state.db"), output: filepath.Join(d, "report.html"), rules: filepath.Join(d, "rules.json")}
	write(t, f.rules, `{"version":1,"rules":[{"name":"Test crawler","id":"bot","pattern":"(?i)bot","enabled":true}]}`)
	return f
}
func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0600); err != nil {
		t.Fatal(err)
	}
}
func event(day, path, ip, ua string, status int) string {
	ts, _ := time.Parse(time.RFC3339, day+"T03:00:00Z")
	x := map[string]any{"ts": float64(ts.Unix()), "status": status, "request": map[string]any{"host": "example.com", "method": "GET", "uri": path, "client_ip": ip, "headers": map[string][]string{"User-Agent": {ua}}}}
	b, _ := json.Marshal(x)
	return string(b) + "\n"
}
func (f fixture) args() []string {
	return []string{"--input", f.log, "--host", "example.com", "--state", f.state, "--output", f.output, "--crawler-rules", f.rules}
}
func execute(t *testing.T, f fixture) {
	t.Helper()
	if err := run(context.Background(), f.args(), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}
func scalar(t *testing.T, f fixture, q string) int64 {
	t.Helper()
	db, err := sql.Open("sqlite", f.state)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int64
	if err = db.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func TestIncrementalUniqueAndPending(t *testing.T) {
	f := setup(t)
	a := event("2026-10-02", "/blogs/1.html?x=1", "192.0.2.1", "Browser", 200)
	b := event("2026-10-01", "/blogs/2.html", "192.0.2.1", "Browser", 304)
	write(t, f.log, a+b[:len(b)/2])
	execute(t, f)
	execute(t, f)
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 1 {
		t.Fatal(n)
	}
	write(t, f.log, a+b+event("2026-10-02", "/blogs/1.html", "192.0.2.2", "Browser", 200))
	execute(t, f)
	execute(t, f)
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 3 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT COUNT(DISTINCT visitor_key) FROM article_daily_visitors"); n != 2 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT SUM(valid_lines) FROM log_generations"); n != 3 {
		t.Fatal(n)
	}
}
func TestRollingGzipAndCopy(t *testing.T) {
	f := setup(t)
	a := event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200)
	b := event("2026-10-02", "/blogs/2.html", "192.0.2.1", "Browser", 200)
	write(t, f.log, a)
	execute(t, f)
	file, err := os.OpenFile(f.log, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString(b)
	file.Close()
	old := filepath.Join(f.dir, "access.log-2026-10-02.json")
	if err = os.Rename(f.log, old); err != nil {
		t.Fatal(err)
	}
	write(t, f.log, b)
	gzPath := old + ".gz"
	out, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	gz.Write([]byte(a + b))
	gz.Close()
	out.Close()
	os.Remove(old)
	args := f.args()
	args = append(args, "--input", gzPath)
	if err = run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err = run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 3 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT COUNT(*) FROM log_generations"); n != 2 {
		t.Fatal(n)
	}
	copyPath := filepath.Join(f.dir, "copy.json")
	write(t, copyPath, a+b)
	args = f.args()
	args[1] = copyPath
	if err = run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 3 {
		t.Fatal(n)
	}
}
func TestCrawler404AndEscaping(t *testing.T) {
	f := setup(t)
	write(t, f.log, event("2026-10-01", "/<script>alert(1)</script>?secret=1", "192.0.2.1", "<script>Bot</script>", 404)+event("2026-10-02", "/blogs/1.html", "", "Browser", 200))
	execute(t, f)
	if n := scalar(t, f, "SELECT SUM(access_count) FROM crawler_daily"); n != 1 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM not_found_daily"); n != 1 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT COUNT(*) FROM article_daily_visitors"); n != 0 {
		t.Fatal(n)
	}
	b, _ := os.ReadFile(f.output)
	s := string(b)
	if strings.Contains(s, "<script>") || strings.Contains(s, "secret=1") || strings.Contains(s, "ZgotmplZ") {
		t.Fatal("unsafe output")
	}
	if !strings.Contains(s, "&lt;script&gt;") {
		t.Fatal("escaping missing")
	}
}
func TestConditionsAndTruncationRejected(t *testing.T) {
	f := setup(t)
	a := event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200)
	write(t, f.log, a)
	execute(t, f)
	write(t, f.rules, `{"version":1,"rules":[{"name":"Test crawler","id":"new","pattern":"crawler","enabled":true}]}`)
	if err := run(context.Background(), f.args(), &bytes.Buffer{}); err == nil {
		t.Fatal("changed rules accepted")
	}
	write(t, f.rules, `{"version":1,"rules":[{"name":"Test crawler","id":"bot","pattern":"(?i)bot","enabled":true}]}`)
	write(t, f.log, a[:len(a)/2])
	if err := run(context.Background(), f.args(), &bytes.Buffer{}); err == nil {
		t.Fatal("truncation accepted")
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 1 {
		t.Fatal(n)
	}
}
func TestOutputFailureRetryAndRanking(t *testing.T) {
	f := setup(t)
	var b strings.Builder
	for i := 1; i <= 25; i++ {
		b.WriteString(event("2026-10-01", fmt.Sprintf("/blogs/%d.html", i), "192.0.2.1", "Browser", 200))
	}
	write(t, f.log, b.String())
	good := f.output
	f.output = filepath.Join(f.dir, "blocked", "index.html")
	write(t, filepath.Join(f.dir, "blocked"), "not a directory")
	if err := run(context.Background(), f.args(), &bytes.Buffer{}); err == nil {
		t.Fatal("output failure expected")
	}
	f.output = good
	execute(t, f)
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 25 {
		t.Fatal(n)
	}
	out, _ := os.ReadFile(good)
	if !strings.Contains(string(out), "残り5記事") {
		t.Fatal("overflow missing")
	}
	if n := scalar(t, f, "SELECT COUNT(DISTINCT visitor_key) FROM article_daily_visitors"); n != 1 {
		t.Fatal(n)
	}
}
func TestMonthStartAndRules(t *testing.T) {
	for _, tc := range []struct{ end, want string }{{"2026-10-15", "2026-09-16"}, {"2026-03-30", "2026-02-28"}, {"2026-02-28", "2026-02-01"}} {
		end, _ := time.Parse("2006-01-02", tc.end)
		if got := monthStart(end).Format("2006-01-02"); got != tc.want {
			t.Fatalf("%s: %s", tc.end, got)
		}
	}
	f := setup(t)
	for _, body := range []string{`{"version":1,"rules":[{"name":"Test crawler","id":"x","pattern":"[","enabled":true}]}`, `{"version":1,"rules":[],"unknown":1}`, `{"version":1,"rules":[{"name":"Test crawler","id":"x","pattern":"bot"}]}`} {
		write(t, f.rules, body)
		o := options{Rules: f.rules}
		if o.loadRules() == nil {
			t.Fatal("bad rules accepted")
		}
	}
}
func TestTemplateCopiedCSS(t *testing.T) {
	mock, err := os.ReadFile("../../docs/mocks/access-report/index.html")
	if err != nil {
		t.Fatal(err)
	}
	css := func(s string) string { return strings.Split(strings.Split(s, "<style>")[1], "</style>")[0] }
	styles, err := reportTemplates.ReadFile("templates/styles.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if css(string(mock)) != css(string(styles)) {
		t.Fatal("mock CSS changed during templating")
	}
}

func TestGzipCorruptionRollsBack(t *testing.T) {
	f := setup(t)
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200))
	execute(t, f)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write([]byte(event("2026-10-02", "/blogs/2.html", "192.0.2.2", "Browser", 200)))
	gz.Close()
	raw := b.Bytes()
	raw[len(raw)-5] ^= 0xff
	bad := filepath.Join(f.dir, "access.log-2026-10-02.json.gz")
	if err := os.WriteFile(bad, raw, 0600); err != nil {
		t.Fatal(err)
	}
	args := f.args()
	args[1] = bad
	if err := run(context.Background(), args, &bytes.Buffer{}); err == nil {
		t.Fatal("corrupt gzip accepted")
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 1 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT COUNT(*) FROM log_generations"); n != 1 {
		t.Fatal(n)
	}
}
func TestInitialPartial(t *testing.T) {
	f := setup(t)
	a := event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200)
	write(t, f.log, a[:len(a)/2])
	if err := run(context.Background(), f.args(), &bytes.Buffer{}); err == nil {
		t.Fatal("no valid lines accepted")
	}
	write(t, f.log, a)
	execute(t, f)
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 1 {
		t.Fatal(n)
	}
}
func TestFullAndIncrementalAgree(t *testing.T) {
	full, delta := setup(t), setup(t)
	lines := []string{event("2026-10-02", "/blogs/1.html", "192.0.2.1", "Browser", 200), event("2026-10-01", "/blogs/2.html", "192.0.2.1", "Browser", 200), event("2026-10-02", "/blogs/1.html", "192.0.2.2", "Bot", 304), event("2026-10-03", "/missing?x=1", "192.0.2.3", "Bot", 404), event("2026-10-04", "/blogs/1.html", "192.0.2.1", "Browser", 200)}
	write(t, full.log, strings.Join(lines, ""))
	execute(t, full)
	for i := range lines {
		write(t, delta.log, strings.Join(lines[:i+1], ""))
		execute(t, delta)
	}
	for _, q := range []string{"SELECT SUM(access_count) FROM article_daily", "SELECT COUNT(DISTINCT visitor_key) FROM article_daily_visitors", "SELECT COUNT(*) FROM article_daily_visitors", "SELECT SUM(access_count) FROM crawler_daily", "SELECT SUM(access_count) FROM not_found_daily"} {
		a, b := scalar(t, full, q), scalar(t, delta, q)
		if a != b {
			t.Fatalf("%s: %d != %d", q, a, b)
		}
	}
}
func TestRuleMetadataDoesNotInvalidate(t *testing.T) {
	f := setup(t)
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Bot", 200))
	execute(t, f)
	write(t, f.rules, `{"rules":[{"enabled":true,"pattern":"(?i)bot","description":"new description","name":"Test crawler","id":"bot"}],"version":1}`)
	execute(t, f)
	if n := scalar(t, f, "SELECT SUM(access_count) FROM crawler_daily"); n != 1 {
		t.Fatal(n)
	}
}

func TestRealTitlesRefreshWithoutReimport(t *testing.T) {
	f := setup(t)
	blogDB := filepath.Join(f.dir, "articles #1.db")
	db, err := sql.Open("sqlite", blogDB)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("CREATE TABLE blogs(id INTEGER PRIMARY KEY,title TEXT NOT NULL); INSERT INTO blogs VALUES(1,'実際の記事タイトル'),(2,'<script>タイトル</script>')"); err != nil {
		t.Fatal(err)
	}
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200)+event("2026-10-01", "/blogs/2.html", "192.0.2.1", "Browser", 200)+event("2026-10-01", "/blogs/3.html", "192.0.2.1", "Browser", 200))
	args := append(f.args(), "--blog-db", blogDB)
	before, err := os.ReadFile(blogDB)
	if err != nil {
		t.Fatal(err)
	}
	if err = run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(blogDB)
	if !bytes.Equal(before, after) {
		t.Fatal("article DB was modified")
	}
	output, _ := os.ReadFile(f.output)
	for _, expected := range []string{"実際の記事タイトル", "&lt;script&gt;タイトル&lt;/script&gt;", "記事 3"} {
		if !strings.Contains(string(output), expected) {
			t.Fatalf("missing title: %s", expected)
		}
	}
	if _, err = db.Exec("UPDATE blogs SET title='更新後の記事タイトル' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err = run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	output, _ = os.ReadFile(f.output)
	if !strings.Contains(string(output), "更新後の記事タイトル") || strings.Contains(string(output), "実際の記事タイトル") {
		t.Fatal("title was not refreshed")
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 3 {
		t.Fatal(n)
	}
}
func TestTitleDBMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if _, err := loadArticleTitles(context.Background(), path, false); err != nil {
		t.Fatal(err)
	}
	if _, err := loadArticleTitles(context.Background(), path, true); err == nil {
		t.Fatal("explicit missing DB accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("missing article DB was created")
	}
}

func TestCrawlersExcludedFromArticleCounts(t *testing.T) {
	f := setup(t)
	browser := event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200)
	bot := event("2026-10-01", "/blogs/1.html", "192.0.2.2", "ExampleBot", 200)
	bot304 := event("2026-10-02", "/blogs/2.html", "192.0.2.3", "ExampleBot", 304)
	bot404 := event("2026-10-02", "/missing", "192.0.2.3", "ExampleBot", 404)
	write(t, f.log, browser+bot)
	execute(t, f)
	write(t, f.log, browser+bot+bot304+bot404)
	execute(t, f)
	execute(t, f)
	checks := map[string]int64{
		"SELECT SUM(access_count) FROM article_daily":                    1,
		"SELECT COUNT(DISTINCT visitor_key) FROM article_daily_visitors": 1,
		"SELECT COUNT(*) FROM articles":                                  1,
		"SELECT SUM(article_excluded_lines) FROM log_generations":        2,
		"SELECT SUM(valid_lines) FROM log_generations":                   4,
		"SELECT SUM(access_count) FROM crawler_daily":                    3,
		"SELECT SUM(access_count) FROM not_found_daily":                  1,
	}
	for q, want := range checks {
		if got := scalar(t, f, q); got != want {
			t.Fatalf("%s: got %d want %d", q, got, want)
		}
	}
	db, err := sql.Open("sqlite", f.state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE report_state SET rules_version=1"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err = run(context.Background(), f.args(), &bytes.Buffer{}); err == nil {
		t.Fatal("old aggregation rules accepted")
	}
}

func TestCrawlerNamesAndFirstMatch(t *testing.T) {
	f := setup(t)
	write(t, f.rules, `{"version":1,"rules":[{"id":"disabled","name":"無効","pattern":"Bot","enabled":false},{"id":"specific","name":"Example crawler","pattern":"(?i)examplebot","enabled":true},{"id":"generic","name":"その他","pattern":"(?i)bot","enabled":true}]}`)
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "ExampleBot/1.0", 200))
	execute(t, f)
	db, err := sql.Open("sqlite", f.state)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err = db.QueryRow("SELECT rule_id FROM crawler_agents").Scan(&id); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if id != "specific" {
		t.Fatalf("first match: %s", id)
	}
	b, _ := os.ReadFile(f.output)
	if !strings.Contains(string(b), "Example crawler") || !strings.Contains(string(b), "ExampleBot/1.0") {
		t.Fatal("crawler name or UA missing")
	}
	original, _ := os.ReadFile(f.rules)
	write(t, f.rules, strings.Replace(string(original), "Example crawler", "<script>新しい名前</script>", 1))
	execute(t, f)
	b, _ = os.ReadFile(f.output)
	if !strings.Contains(string(b), "&lt;script&gt;新しい名前&lt;/script&gt;") || strings.Contains(string(b), "Example crawler") {
		t.Fatal("name refresh or escaping failed")
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM crawler_daily"); n != 1 {
		t.Fatal(n)
	}
	if n := scalar(t, f, "SELECT COUNT(*) FROM article_daily"); n != 0 {
		t.Fatal(n)
	}
	write(t, f.rules, `{"version":1,"rules":[{"id":"generic","name":"その他","pattern":"(?i)bot","enabled":true},{"id":"specific","name":"Example crawler","pattern":"(?i)examplebot","enabled":true}]}`)
	if err = run(context.Background(), f.args(), &bytes.Buffer{}); err == nil {
		t.Fatal("changed priority accepted")
	}
}
func TestCrawlerNameRequired(t *testing.T) {
	f := setup(t)
	for _, name := range []string{"", "  "} {
		write(t, f.rules, `{"version":1,"rules":[{"id":"bot","name":"`+name+`","pattern":"bot","enabled":true}]}`)
		o := options{Rules: f.rules}
		if o.loadRules() == nil {
			t.Fatal("empty name accepted")
		}
	}
}

func TestScanCandidatesByIPAndDay(t *testing.T) {
	f := setup(t)
	var lines strings.Builder
	for i := 0; i < 100; i++ {
		ua := "Browser"
		if i%2 == 0 {
			ua = "ExampleBot"
		}
		lines.WriteString(event("2026-10-01", fmt.Sprintf("/probe/%d?token=%d", i%20, i), "192.0.2.10", ua, 404))
		lines.WriteString(event("2026-10-01", "/single", "192.0.2.11", "Browser", 404))
		lines.WriteString(event("2026-10-01", fmt.Sprintf("/missing/%d", i), "", "Browser", 404))
	}
	for i := 0; i < 99; i++ {
		lines.WriteString(event("2026-10-01", fmt.Sprintf("/below/%d", i), "192.0.2.12", "Browser", 404))
	}
	for _, day := range []string{"2026-10-01", "2026-10-02"} {
		for i := 0; i < 50; i++ {
			lines.WriteString(event(day, fmt.Sprintf("/split/%d", i), "192.0.2.13", "Browser", 404))
		}
	}
	write(t, f.log, lines.String())
	execute(t, f)
	execute(t, f)
	db, err := sql.Open("sqlite", f.state)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := state{db: db}
	from, _ := time.Parse("2006-01-02", "2026-10-01")
	to := from.AddDate(0, 0, 1)
	o := options{loc: time.UTC}
	candidates, err := st.scans(context.Background(), from, to, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates: %+v", candidates)
	}
	r := candidates[0]
	if r.IP != "192.0.2.10" || r.Count != 100 || r.PathCount != 20 || r.DetectedDays != 1 || r.Peak != 100 {
		t.Fatalf("wrong candidate: %+v", r)
	}
	if len(r.Agents) != 2 || len(r.Paths) != 5 {
		t.Fatalf("missing evidence: %+v", r)
	}
	if got := scalar(t, f, "SELECT SUM(access_count) FROM scan_daily_paths"); got != 399 {
		t.Fatal(got)
	}
	html, _ := os.ReadFile(f.output)
	if !strings.Contains(string(html), "脆弱性スキャンの疑い") || !strings.Contains(string(html), "192.0.2.10") {
		t.Fatal("candidate missing from HTML")
	}
	if strings.Contains(string(html), "token=") {
		t.Fatal("query leaked")
	}
	if candidates, err = st.scans(context.Background(), to, to, o); err != nil || len(candidates) != 0 {
		t.Fatalf("period filter: %+v %v", candidates, err)
	}

}
