package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type accessLog struct {
	TS      *float64 `json:"ts"`
	Status  int      `json:"status"`
	Request struct {
		Host     string              `json:"host"`
		Method   string              `json:"method"`
		URI      string              `json:"uri"`
		ClientIP string              `json:"client_ip"`
		RemoteIP string              `json:"remote_ip"`
		Headers  map[string][]string `json:"headers"`
	} `json:"request"`
}

var articlePattern = regexp.MustCompile(`^/blogs/([0-9]+)\.html$`)

func normalizeHost(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if host, port, err := net.SplitHostPort(s); err == nil && (port == "443" || port == "80") {
		s = host
	}
	return strings.TrimSuffix(s, ".")
}
func normalizedIP(s string) string {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return ""
	}
	return ip.String()
}
func (o options) crawlerID(ua string) string {
	if ua == "" {
		return ""
	}
	for _, r := range o.compiledRules {
		if r.Regex.MatchString(ua) {
			return r.ID
		}
	}
	return ""
}
func display(s string, max int) (string, int) {
	rs := []rune(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s))
	if len(rs) > max {
		return string(rs[:max]) + "", 1
	}
	return string(rs), 0
}

type snapshot struct {
	r    io.Reader
	file *os.File
	gz   *gzip.Reader
	info os.FileInfo
}

func readSnapshot(path string) (*snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("通常ファイルではありません")
	}
	sn := &snapshot{file: f, info: info, r: io.LimitReader(f, info.Size())}
	if strings.HasSuffix(path, ".gz") {
		g, err := gzip.NewReader(sn.r)
		if err != nil {
			f.Close()
			return nil, err
		}
		sn.gz = g
		sn.r = g
	}
	return sn, nil
}
func (s *snapshot) close() {
	if s.gz != nil {
		s.gz.Close()
	}
	s.file.Close()
}
func prefixHash(path string, n int64) ([]byte, error) {
	s, err := readSnapshot(path)
	if err != nil {
		return nil, err
	}
	defer s.close()
	h := sha256.New()
	_, err = io.CopyN(h, s.r, n)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

type generation struct {
	id, offset int64
	hash       []byte
	sealed     int
	headLength int64
	headHash   []byte
}

// Capture a bounded prefix while the original snapshot is read, including reads
// used to verify the committed digest. Do not retain log lines beyond this limit.
type prefixCapture struct{ data []byte }

func (p *prefixCapture) Write(b []byte) (int, error) {
	n := len(b)
	remaining := 4096 - len(p.data)
	if remaining > len(b) {
		remaining = len(b)
	}
	if remaining > 0 {
		p.data = append(p.data, b[:remaining]...)
	}
	return n, nil
}

func (st *state) ingest(ctx context.Context, path string, o options) (int64, int64, error) {
	sn, err := readSnapshot(path)
	if err != nil {
		return 0, 0, err
	}
	defer sn.close()
	device, inode := fileIdentity(sn.info)
	var physicalID sql.NullInt64
	if device != "" {
		if err = st.db.QueryRowContext(ctx, "SELECT generation_id FROM log_files WHERE device=? AND inode=? ORDER BY id DESC LIMIT 1", device, inode).Scan(&physicalID); err != nil && err != sql.ErrNoRows {
			return 0, 0, err
		}
	}
	rows, err := st.db.QueryContext(ctx, "SELECT id,committed_offset,committed_sha256,sealed,head_length,head_sha256 FROM log_generations WHERE committed_offset>0 ORDER BY committed_offset")
	if err != nil {
		return 0, 0, err
	}
	var gens []generation
	for rows.Next() {
		var g generation
		if err = rows.Scan(&g.id, &g.offset, &g.hash, &g.sealed, &g.headLength, &g.headHash); err != nil {
			rows.Close()
			return 0, 0, err
		}
		gens = append(gens, g)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, 0, err
	}
	var matches []generation
	hashes := map[int64][]byte{}
	heads := map[int64][]byte{}
	short := map[int64]bool{}
	for _, g := range gens {
		if sn.gz == nil && g.offset > sn.info.Size() {
			continue
		}
		if g.headLength > 0 {
			candidate, ok := heads[g.headLength]
			if !ok {
				candidate, err = prefixHash(path, g.headLength)
				if err != nil && !errors.Is(err, io.EOF) {
					return 0, 0, err
				}
				heads[g.headLength] = candidate
			}
			if !bytes.Equal(candidate, g.headHash) {
				continue
			}
		}
		h, ok := hashes[g.offset]
		if !ok && !short[g.offset] {
			h, err = prefixHash(path, g.offset)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				short[g.offset] = true
			} else if err != nil {
				return 0, 0, err
			} else {
				hashes[g.offset] = h
			}
		}
		if bytes.Equal(h, g.hash) {
			matches = append(matches, g)
		}
	}
	if len(matches) > 1 {
		return 0, 0, errors.New("ログ世代の照合が曖昧です")
	}
	g := generation{}
	if len(matches) == 1 {
		g = matches[0]
	}
	if physicalID.Valid && g.id == 0 {
		var emptyOffset int64
		if err = st.db.QueryRowContext(ctx, "SELECT committed_offset FROM log_generations WHERE id=?", physicalID.Int64).Scan(&emptyOffset); err != nil {
			return 0, 0, err
		}
		if emptyOffset == 0 {
			g.id = physicalID.Int64
		}
	}
	if physicalID.Valid && physicalID.Int64 != g.id {
		return 0, 0, errors.New("同じファイルの内容変更・切り詰めを検出しました")
	}
	// A previously known path may now contain a new generation after rename.
	// Identity changes are resolved by full committed-prefix comparison above.
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	now := stamp()
	if g.id == 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO log_generations(created_at,updated_at) VALUES(?,?)", now, now)
		if err != nil {
			return 0, 0, err
		}
		g.id, err = res.LastInsertId()
		if err != nil {
			return 0, 0, err
		}
	}
	var captured prefixCapture
	sn.r = io.TeeReader(sn.r, &captured)
	h := sha256.New()
	if _, err = io.CopyN(h, sn.r, g.offset); err != nil {
		return 0, 0, err
	}
	if g.offset > 0 && !bytes.Equal(h.Sum(nil), g.hash) {
		return 0, 0, errors.New("読み込み中にログ内容が変わりました")
	}
	br := bufio.NewReaderSize(sn.r, 64*1024)
	offset := g.offset
	var processed, valid, invalid, hostOut, article, excluded, other, unknown, oversized int64
	var first, last *int64
	pending := false
	for {
		if err = ctx.Err(); err != nil {
			return 0, 0, err
		}
		line, readErr := readLine(br, 16*1024*1024)
		if readErr != nil && readErr != io.EOF {
			return 0, 0, readErr
		}
		if len(line) > 0 && line[len(line)-1] != '\n' {
			pending = true
			break
		}
		if len(line) == 0 {
			break
		}
		if g.sealed != 0 {
			return 0, 0, errors.New("完了済みのログ世代に追記があります")
		}
		h.Write(line)
		offset += int64(len(line))
		processed++
		var x accessLog
		if json.Unmarshal(line, &x) != nil || x.TS == nil || math.IsNaN(*x.TS) || math.IsInf(*x.TS, 0) || *x.TS < 0 || *x.TS >= 253402300800 || x.Status < 100 || x.Status > 599 || x.Request.Host == "" || x.Request.Method == "" || x.Request.URI == "" {
			invalid++
			continue
		}
		valid++
		r := x.Request
		if normalizeHost(r.Host) != o.Host {
			hostOut++
			continue
		}
		us := int64(*x.TS * 1e6)
		if first == nil || us < *first {
			v := us
			first = &v
		}
		if last == nil || us > *last {
			v := us
			last = &v
		}
		day := time.UnixMicro(us).In(o.loc).Format("2006-01-02")
		pathOnly := strings.SplitN(r.URI, "?", 2)[0]
		if u, parseErr := url.ParseRequestURI(r.URI); parseErr == nil && u.IsAbs() {
			pathOnly = u.EscapedPath()
			if pathOnly == "" {
				pathOnly = "/"
			}
		}
		var uaParts []string
		var uaKeys []string
		for k := range r.Headers {
			if strings.EqualFold(k, "User-Agent") {
				uaKeys = append(uaKeys, k)
			}
		}
		sort.Strings(uaKeys)
		for _, k := range uaKeys {
			uaParts = append(uaParts, r.Headers[k]...)
		}
		ua := strings.Join(uaParts, " ")
		oversizedRecord := false
		ruleID := o.crawlerID(ua)
		isCrawler := ruleID != ""
		if isCrawler {
			label, tr := display(ua, 1024)
			if tr != 0 && !oversizedRecord {
				oversized++
				oversizedRecord = true
			}
			if err = addDimension(ctx, tx, "crawler", day, ua, label, tr, ruleID); err != nil {
				return 0, 0, err
			}
		}
		if x.Status == 404 {
			label, tr := display(pathOnly, 2048)
			if tr != 0 && !oversizedRecord {
				oversized++
				oversizedRecord = true
			}
			if err = addDimension(ctx, tx, "not_found", day, pathOnly, label, tr, ""); err != nil {
				return 0, 0, err
			}
			ip := normalizedIP(r.ClientIP)
			if ip == "" {
				ip = normalizedIP(r.RemoteIP)
			}
			if ip != "" {
				if err = addScanAccess(ctx, tx, day, ip, pathOnly, ua, ruleID, us); err != nil {
					return 0, 0, err
				}
			}
		}
		m := articlePattern.FindStringSubmatch(pathOnly)
		var id int64
		if len(m) > 0 {
			var parseErr error
			id, parseErr = strconv.ParseInt(m[1], 10, 64)
			if parseErr != nil {
				id = 0
			}
		}
		if id <= 0 || r.Method != "GET" || (x.Status != 200 && x.Status != 304) {
			other++
			continue
		}
		if isCrawler {
			excluded++
			continue
		}
		article++
		ip := normalizedIP(r.ClientIP)
		if ip == "" {
			ip = normalizedIP(r.RemoteIP)
		}
		missing := 0
		if ip == "" {
			missing = 1
			unknown++
		}
		canonical := fmt.Sprintf("/blogs/%d.html", id)
		if _, err = tx.ExecContext(ctx, "INSERT INTO articles(article_id,path) VALUES(?,?) ON CONFLICT DO NOTHING", id, canonical); err != nil {
			return 0, 0, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO article_daily VALUES(?,?,1,?) ON CONFLICT(day,article_id) DO UPDATE SET access_count=access_count+1,unidentified_count=unidentified_count+excluded.unidentified_count", day, id, missing); err != nil {
			return 0, 0, err
		}
		if missing == 0 {
			b, _ := json.Marshal([]string{ip, ua})
			mac := hmac.New(sha256.New, st.key)
			mac.Write(b)
			if _, err = tx.ExecContext(ctx, "INSERT INTO article_daily_visitors VALUES(?,?,?) ON CONFLICT DO NOTHING", day, id, mac.Sum(nil)); err != nil {
				return 0, 0, err
			}
		}
		if readErr == io.EOF {
			break
		}
	}
	after, err := sn.file.Stat()
	if err != nil {
		return 0, 0, err
	}
	if after.Size() < sn.info.Size() {
		return 0, 0, errors.New("読み込み中にログが切り詰められました")
	}
	sealed := 0
	if !pending && (strings.HasSuffix(path, ".gz") || strings.HasPrefix(filepath.Base(path), strings.TrimSuffix(o.LogName, filepath.Ext(o.LogName))+"-")) {
		sealed = 1
	}
	if pending {
		fmt.Fprintln(st.log, "保留: 改行のない末尾:", filepath.Base(path))
	}
	if invalid > 0 {
		fmt.Fprintf(st.log, "警告: %s の不正行 %d件をスキップ\n", filepath.Base(path), invalid)
	}
	var full []byte
	if offset > 0 {
		full = h.Sum(nil)
	}
	headLength := min(offset, int64(len(captured.data)))
	var headDigest []byte
	if headLength > 0 {
		headDigest = digest(captured.data[:headLength])
	}
	var minVal, maxVal any
	if first != nil {
		minVal = *first
		maxVal = *last
	}
	_, err = tx.ExecContext(ctx, `UPDATE log_generations SET committed_offset=?,committed_sha256=?,sealed=?,valid_lines=valid_lines+?,invalid_lines=invalid_lines+?,host_out_lines=host_out_lines+?,article_access_lines=article_access_lines+?,article_excluded_lines=article_excluded_lines+?,article_other_lines=article_other_lines+?,unidentified_article_lines=unidentified_article_lines+?,oversized_dimension_lines=oversized_dimension_lines+?,first_ts_us=CASE WHEN ? IS NULL THEN first_ts_us WHEN first_ts_us IS NULL THEN ? ELSE MIN(first_ts_us,?) END,last_ts_us=CASE WHEN ? IS NULL THEN last_ts_us WHEN last_ts_us IS NULL THEN ? ELSE MAX(last_ts_us,?) END,updated_at=? WHERE id=?`, offset, full, sealed, valid, invalid, hostOut, article, excluded, other, unknown, oversized, minVal, minVal, minVal, maxVal, maxVal, maxVal, now, g.id)
	if err != nil {
		return 0, 0, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE log_generations SET head_length=?,head_sha256=? WHERE id=?", headLength, headDigest, g.id); err != nil {
		return 0, 0, err
	}
	compression := "none"
	if sn.gz != nil {
		compression = "gzip"
	}
	var fileID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM log_files WHERE generation_id=? AND path=? AND COALESCE(device,'')=? AND COALESCE(inode,'')=? LIMIT 1", g.id, path, device, inode).Scan(&fileID)
	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx, "INSERT INTO log_files(generation_id,path,compression,device,inode,observed_size,first_seen_at,last_seen_at) VALUES(?,?,?,?,?,?,?,?)", g.id, path, compression, device, inode, after.Size(), now, now)
	} else if err == nil {
		_, err = tx.ExecContext(ctx, "UPDATE log_files SET observed_size=?,last_seen_at=? WHERE id=?", after.Size(), now, fileID)
	}
	if err != nil {
		return 0, 0, err
	}
	if processed > 0 {
		if _, err = tx.ExecContext(ctx, "UPDATE report_state SET revision=revision+1,updated_at=? WHERE id=1", now); err != nil {
			return 0, 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return processed, g.id, nil
}
func readLine(br *bufio.Reader, max int) ([]byte, error) {
	var line []byte
	for {
		b, err := br.ReadSlice('\n')
		if len(line)+len(b) > max {
			return nil, errors.New("ログ1行が16MiBを超えています")
		}
		line = append(line, b...)
		if err != bufio.ErrBufferFull {
			return line, err
		}
	}
}
func addDimension(ctx context.Context, tx *sql.Tx, kind, day, raw, label string, tr int, ruleID string) error {
	master, daily, key, col, idcol := "crawler_agents", "crawler_daily", "ua_hash", "display_ua", "agent_id"
	if kind == "not_found" {
		master, daily, key, col, idcol = "not_found_paths", "not_found_daily", "path_hash", "display_path", "path_id"
	}
	sum := digest([]byte(raw))
	if tr != 0 {
		rs := []rune(label)
		max := 1024
		if kind == "not_found" {
			max = 2048
		}
		label = string(rs[:max-16]) + "… [" + fmt.Sprintf("%x", sum[:5]) + "]"
	}
	columns, values := key+","+col+",is_truncated", "?,?,?"
	args := []any{sum, label, tr}
	if kind == "crawler" {
		columns += ",rule_id"
		values += ",?"
		args = append(args, ruleID)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO "+master+"("+columns+") VALUES("+values+") ON CONFLICT DO NOTHING", args...); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO "+daily+"(day,"+idcol+",access_count) SELECT ?,id,1 FROM "+master+" WHERE "+key+"=? ON CONFLICT(day,"+idcol+") DO UPDATE SET access_count=access_count+1", day, sum)
	return err
}
