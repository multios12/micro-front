package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

//go:embed schema.sql
var schema string

const aggregationRulesVersion = 2
const stateSchemaVersion = 3

type state struct {
	db  *sql.DB
	key []byte
	log io.Writer
}

func digest(b []byte) []byte { h := sha256.Sum256(b); return h[:] }
func stamp() string          { return time.Now().UTC().Format(time.RFC3339Nano) }
func openState(ctx context.Context, o options) (*state, error) {
	db, err := sql.Open("sqlite", o.State)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ok := false
	defer func() {
		if !ok {
			db.Close()
		}
	}()
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys=ON;"); err != nil {
		return nil, err
	}
	var exists int
	if err = db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='report_state'").Scan(&exists); err != nil {
		return nil, err
	}
	type ruleIdentity struct{ ID, Pattern string }
	identities := make([]ruleIdentity, 0, len(o.compiledRules))
	for _, r := range o.compiledRules {
		identities = append(identities, ruleIdentity{r.ID, r.Pattern})
	}
	config, _ := json.Marshal(struct {
		Host, Timezone string
		Rules          []string
		Version        int
		CrawlerRules   []ruleIdentity
	}{o.Host, reportTimezone, o.patternsText, aggregationRulesVersion, identities})
	hash := digest(config)
	keyPath := filepath.Join(filepath.Dir(o.State), filepath.Base(o.State)+".visitor.key")
	key, err := os.ReadFile(keyPath)
	if errors.Is(err, os.ErrNotExist) && exists == 0 {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		if err = writeAtomic(keyPath, key, 0600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("ユニーク判定鍵を読み込めません: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("ユニーク判定鍵が不正です")
	}
	keyID := hex.EncodeToString(digest(key))
	if exists == 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, schema); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO report_state VALUES(1,?,?,?,?,?,?,?,0,?,?)", stateSchemaVersion, aggregationRulesVersion, o.Host, reportTimezone, string(config), hash, keyID, stamp(), stamp()); err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
	} else {
		var sv, rv int
		var oldHash []byte
		var oldKey string
		if err = db.QueryRowContext(ctx, "SELECT schema_version,rules_version,config_hash,visitor_key_id FROM report_state WHERE id=1").Scan(&sv, &rv, &oldHash, &oldKey); err != nil {
			return nil, err
		}
		if sv != stateSchemaVersion || rv != aggregationRulesVersion || !hmac.Equal(hash, oldHash) || oldKey != keyID {
			return nil, errors.New("スキーマ・集計条件または識別鍵が異なります。別の --state で全件再集計してください")
		}
	}
	if err = os.Chmod(o.State, 0600); err != nil {
		return nil, err
	}
	ok = true
	return &state{db: db, key: key, log: os.Stderr}, nil
}
func writeAtomic(path string, b []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".access-report-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	return os.Rename(f.Name(), path)
}
