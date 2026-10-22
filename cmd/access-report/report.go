package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/*.tmpl
var reportTemplates embed.FS

type dayRow struct {
	Day, Tick                  string
	Accesses, Unique           int64
	AccessHeight, UniqueHeight float64
}
type articleRow struct {
	ID, Accesses, Unique int64
	Path                 string
	Title                string
	Rank                 int
	Share                float64
}
type countRow struct {
	Label, Name, RuleID string
	Count               int64
}
type periodReport struct {
	From, To                                           string
	ObservedFrom, ObservedTo                           string
	Accesses, Unique, ArticleCount                     int64
	Average                                            float64
	DayCount                                           int
	Peak                                               int64
	Days                                               []dayRow
	Top, More                                          []articleRow
	CrawlerTop, CrawlerMore, NotFoundTop, NotFoundMore []countRow
	CrawlerTotal, NotFoundTotal                        int64
	CrawlerCount, NotFoundCount                        int
	ScanTop, ScanMore                                  []scanRow
	ScanCount                                          int
}
type reportData struct {
	Host, Timezone, RulesLabel string
	Revision                   int64
	Month, All                 periodReport
	ScanMin404, ScanMinPaths   int
}

func number(v any) string {
	s := fmt.Sprint(v)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
func monthStart(end time.Time) time.Time {
	until := end.AddDate(0, 0, 1)
	first := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, until.Location()).AddDate(0, -1, 0)
	day := until.Day()
	last := first.AddDate(0, 1, 0).AddDate(0, 0, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, until.Location())
}
func (st *state) render(ctx context.Context, o options) error {
	var min, max *int64
	if err := st.db.QueryRowContext(ctx, "SELECT MIN(first_ts_us),MAX(last_ts_us) FROM log_generations").Scan(&min, &max); err != nil {
		return err
	}
	if min == nil {
		return fmt.Errorf("対象ホストの有効ログがありません")
	}
	start := time.UnixMicro(*min).In(o.loc)
	end := time.UnixMicro(*max).In(o.loc)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, o.loc)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, o.loc)
	if start.After(end) {
		return fmt.Errorf("表示期間内のログがありません")
	}
	if end.Year()-start.Year() > 100 {
		return fmt.Errorf("表示期間が100年を超えています")
	}
	month := monthStart(end)
	all, err := st.period(ctx, start, end)
	if err != nil {
		return err
	}
	recent, err := st.period(ctx, month, end)
	if err != nil {
		return err
	}
	for _, period := range []struct {
		from time.Time
		data *periodReport
	}{{start, &all}, {month, &recent}} {
		scans, err := st.scans(ctx, period.from, end, o)
		if err != nil {
			return err
		}
		period.data.ScanCount = len(scans)
		cut := len(scans)
		if cut > 20 {
			cut = 20
		}
		period.data.ScanTop, period.data.ScanMore = scans[:cut], scans[cut:]
	}
	titles, err := loadArticleTitles(ctx, o.BlogDB, o.blogDBExplicit)
	if err != nil {
		return err
	}
	for _, rows := range [][]articleRow{all.Top, all.More, recent.Top, recent.More} {
		for i := range rows {
			rows[i].Title = titles[rows[i].ID]
			if strings.TrimSpace(rows[i].Title) == "" {
				rows[i].Title = fmt.Sprintf("記事 %d", rows[i].ID)
			}
		}
	}
	names := make(map[string]string, len(o.compiledRules))
	for _, rule := range o.compiledRules {
		names[rule.ID] = rule.Name
	}
	for _, rows := range [][]countRow{all.CrawlerTop, all.CrawlerMore, recent.CrawlerTop, recent.CrawlerMore} {
		for i := range rows {
			rows[i].Name = names[rows[i].RuleID]
		}
	}
	for _, scans := range [][]scanRow{all.ScanTop, all.ScanMore, recent.ScanTop, recent.ScanMore} {
		for i := range scans {
			for j := range scans[i].Agents {
				scans[i].Agents[j].Name = names[scans[i].Agents[j].RuleID]
			}
		}
	}
	data := reportData{Host: o.Host, Timezone: reportTimezone, RulesLabel: strings.Join(o.patternsText, " / "), All: all, Month: recent, ScanMin404: scanMin404, ScanMinPaths: scanMinPaths}
	if data.RulesLabel == "" {
		data.RulesLabel = "無効"
	}
	if err = st.db.QueryRowContext(ctx, "SELECT revision FROM report_state WHERE id=1").Scan(&data.Revision); err != nil {
		return err
	}
	t, err := template.New("report").Funcs(template.FuncMap{"number": number}).ParseFS(reportTemplates, "templates/*.tmpl")
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err = t.ExecuteTemplate(&b, "report.tmpl", data); err != nil {
		return err
	}
	return writeAtomic(o.Output, b.Bytes(), 0644)
}
func (st *state) period(ctx context.Context, from, to time.Time) (periodReport, error) {
	p := periodReport{From: from.Format("2006 / 01 / 02"), To: to.Format("2006 / 01 / 02"), ObservedFrom: "該当なし", ObservedTo: "該当なし"}
	a, b := from.Format("2006-01-02"), to.AddDate(0, 0, 1).Format("2006-01-02")
	args := []any{a, b}
	var min, max *string
	if err := st.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(access_count),0),COUNT(DISTINCT article_id),MIN(day),MAX(day) FROM article_daily WHERE day>=? AND day<?", args...).Scan(&p.Accesses, &p.ArticleCount, &min, &max); err != nil {
		return p, err
	}
	if min != nil {
		p.ObservedFrom = strings.ReplaceAll(*min, "-", " / ")
		p.ObservedTo = strings.ReplaceAll(*max, "-", " / ")
	}
	if err := st.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT visitor_key) FROM article_daily_visitors WHERE day>=? AND day<?", args...).Scan(&p.Unique); err != nil {
		return p, err
	}
	daily := map[string]*dayRow{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		daily[key] = &dayRow{Day: key}
		p.Days = append(p.Days, dayRow{Day: key})
	}
	p.DayCount = len(p.Days)
	for _, spec := range []struct {
		sql    string
		unique bool
	}{{"SELECT day,SUM(access_count) FROM article_daily WHERE day>=? AND day<? GROUP BY day", false}, {"SELECT day,COUNT(DISTINCT visitor_key) FROM article_daily_visitors WHERE day>=? AND day<? GROUP BY day", true}} {
		rows, err := st.db.QueryContext(ctx, spec.sql, args...)
		if err != nil {
			return p, err
		}
		for rows.Next() {
			var d string
			var n int64
			if err = rows.Scan(&d, &n); err != nil {
				rows.Close()
				return p, err
			}
			if spec.unique {
				daily[d].Unique = n
			} else {
				daily[d].Accesses = n
				if n > p.Peak {
					p.Peak = n
				}
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return p, err
		}
	}
	for i := range p.Days {
		d := *daily[p.Days[i].Day]
		if p.Peak > 0 {
			d.AccessHeight = float64(d.Accesses) / float64(p.Peak) * 100
			d.UniqueHeight = float64(d.Unique) / float64(p.Peak) * 100
		}
		if i == 0 || i == len(p.Days)-1 || strings.HasSuffix(d.Day, "-01") {
			d.Tick = strings.ReplaceAll(d.Day[5:], "-", "/")
		}
		p.Days[i] = d
	}
	if p.DayCount > 0 {
		p.Average = float64(p.Accesses) / float64(p.DayCount)
	}
	rows, err := st.db.QueryContext(ctx, `WITH c AS (SELECT article_id,SUM(access_count) AS n FROM article_daily WHERE day>=? AND day<? GROUP BY article_id),v AS (SELECT article_id,COUNT(DISTINCT visitor_key) AS n FROM article_daily_visitors WHERE day>=? AND day<? GROUP BY article_id) SELECT a.article_id,a.path,c.n,COALESCE(v.n,0) FROM c JOIN articles a ON a.article_id=c.article_id LEFT JOIN v ON v.article_id=c.article_id ORDER BY c.n DESC,a.article_id`, a, b, a, b)
	if err != nil {
		return p, err
	}
	i := 0
	for rows.Next() {
		var r articleRow
		if err = rows.Scan(&r.ID, &r.Path, &r.Accesses, &r.Unique); err != nil {
			rows.Close()
			return p, err
		}
		i++
		r.Rank = i
		if p.Accesses > 0 {
			r.Share = float64(r.Accesses) / float64(p.Accesses) * 100
		}
		if i <= 20 {
			p.Top = append(p.Top, r)
		} else {
			p.More = append(p.More, r)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return p, err
	}
	for _, spec := range []struct {
		query   string
		crawler bool
	}{{`SELECT a.display_ua,SUM(d.access_count) AS n,a.rule_id FROM crawler_daily d JOIN crawler_agents a ON a.id=d.agent_id WHERE day>=? AND day<? GROUP BY a.id ORDER BY n DESC,a.display_ua,a.id`, true}, {`SELECT a.display_path,SUM(d.access_count) AS n,'' FROM not_found_daily d JOIN not_found_paths a ON a.id=d.path_id WHERE day>=? AND day<? GROUP BY a.id ORDER BY n DESC,a.display_path,a.id`, false}} {
		rows, err := st.db.QueryContext(ctx, spec.query, args...)
		if err != nil {
			return p, err
		}
		var list []countRow
		for rows.Next() {
			var r countRow
			if err = rows.Scan(&r.Label, &r.Count, &r.RuleID); err != nil {
				rows.Close()
				return p, err
			}
			list = append(list, r)
			if spec.crawler {
				p.CrawlerTotal += r.Count
			} else {
				p.NotFoundTotal += r.Count
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return p, err
		}
		n := len(list)
		cut := n
		if cut > 20 {
			cut = 20
		}
		if spec.crawler {
			p.CrawlerCount = n
			p.CrawlerTop = list[:cut]
			p.CrawlerMore = list[cut:]
		} else {
			p.NotFoundCount = n
			p.NotFoundTop = list[:cut]
			p.NotFoundMore = list[cut:]
		}
	}
	return p, nil
}
