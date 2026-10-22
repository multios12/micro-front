package main

import (
	"context"
	"database/sql"
	"time"
)

// Store 404 evidence independently of the UA classification. A source remains
// the same when it changes its User-Agent. Missing/invalid IPs are not merged.
func addScanAccess(ctx context.Context, tx *sql.Tx, day, ip, path, ua, ruleID string, us int64) error {
	if _, err := tx.ExecContext(ctx, "INSERT INTO scan_sources(ip) VALUES(?) ON CONFLICT DO NOTHING", ip); err != nil {
		return err
	}
	var sourceID, pathID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM scan_sources WHERE ip=?", ip).Scan(&sourceID); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, "SELECT id FROM not_found_paths WHERE path_hash=?", digest([]byte(path))).Scan(&pathID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO scan_daily_paths VALUES(?,?,?,1,?,?) ON CONFLICT(day,source_id,path_id) DO UPDATE SET access_count=access_count+1,first_ts_us=MIN(first_ts_us,excluded.first_ts_us),last_ts_us=MAX(last_ts_us,excluded.last_ts_us)`, day, sourceID, pathID, us, us); err != nil {
		return err
	}
	label, tr := display(ua, 1024)
	if tr != 0 {
		label = string([]rune(label)[:1023]) + "…"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO scan_daily_agents VALUES(?,?,?,?,?,1) ON CONFLICT(day,source_id,ua_hash) DO UPDATE SET access_count=access_count+1`, day, sourceID, digest([]byte(ua)), label, ruleID)
	return err
}

type scanRow struct {
	ID                                   int64
	IP                                   string
	Count, PathCount, DetectedDays, Peak int64
	First, Last                          string
	Paths, Agents                        []countRow
}

func (st *state) scans(ctx context.Context, from, to time.Time, o options) ([]scanRow, error) {
	a, b := from.Format("2006-01-02"), to.AddDate(0, 0, 1).Format("2006-01-02")
	rows, err := st.db.QueryContext(ctx, `WITH daily AS (
 SELECT day,source_id,SUM(access_count) AS n,COUNT(*) AS paths
 FROM scan_daily_paths WHERE day>=? AND day<? GROUP BY day,source_id
 ), detected AS (
 SELECT source_id,COUNT(*) AS days,MAX(n) AS peak FROM daily
 WHERE n>=? AND paths>=? GROUP BY source_id
 )
 SELECT s.id,s.ip,SUM(p.access_count) AS n,COUNT(DISTINCT p.path_id),d.days,d.peak,MIN(p.first_ts_us),MAX(p.last_ts_us)
 FROM detected d JOIN scan_sources s ON s.id=d.source_id JOIN scan_daily_paths p ON p.source_id=s.id
 WHERE p.day>=? AND p.day<? GROUP BY s.id
 ORDER BY n DESC,s.ip ASC`, a, b, scanMin404, scanMinPaths, a, b)
	if err != nil {
		return nil, err
	}
	var result []scanRow
	for rows.Next() {
		var r scanRow
		var first, last int64
		if err = rows.Scan(&r.ID, &r.IP, &r.Count, &r.PathCount, &r.DetectedDays, &r.Peak, &first, &last); err != nil {
			rows.Close()
			return nil, err
		}
		r.First = time.UnixMicro(first).In(o.loc).Format("2006-01-02 15:04:05")
		r.Last = time.UnixMicro(last).In(o.loc).Format("2006-01-02 15:04:05")
		result = append(result, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range result {
		for _, agents := range []bool{false, true} {
			query := `SELECT p.display_path,SUM(d.access_count) AS n,'' FROM scan_daily_paths d JOIN not_found_paths p ON p.id=d.path_id WHERE d.source_id=? AND d.day>=? AND d.day<? GROUP BY p.id ORDER BY n DESC,p.display_path,p.id LIMIT 5`
			if agents {
				query = `SELECT display_ua,SUM(access_count) AS n,rule_id FROM scan_daily_agents WHERE source_id=? AND day>=? AND day<? GROUP BY ua_hash ORDER BY n DESC,display_ua LIMIT 5`
			}
			rows, err := st.db.QueryContext(ctx, query, result[i].ID, a, b)
			if err != nil {
				return nil, err
			}
			var list []countRow
			for rows.Next() {
				var r countRow
				if err = rows.Scan(&r.Label, &r.Count, &r.RuleID); err != nil {
					rows.Close()
					return nil, err
				}
				if agents && r.Label == "" {
					r.Label = "（User-Agentなし）"
				}
				list = append(list, r)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, err
			}
			if agents {
				result[i].Agents = list
			} else {
				result[i].Paths = list
			}
		}
	}
	return result, nil
}
