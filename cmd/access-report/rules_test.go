package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMissingRulesCreatedFromDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "crawlers.json")
	o := options{Rules: path}
	if err := o.loadRules(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, defaultCrawlerRules) {
		t.Fatal("created rules differ from default")
	}
	if len(o.compiledRules) == 0 {
		t.Fatal("default rules were not loaded")
	}
}

func TestExistingRulesNotOverwritten(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"custom", `{"version":1,"rules":[]}`, false},
		{"invalid", `invalid JSON`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "crawlers.json")
			write(t, path, tc.body)
			o := options{Rules: path}
			if err := o.loadRules(); (err != nil) != tc.wantError {
				t.Fatalf("loadRules: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.body {
				t.Fatal("existing rules overwritten")
			}
			if err := createDefaultRules(path); err != nil {
				t.Fatal(err)
			}
			got, err = os.ReadFile(path)
			if err != nil || string(got) != tc.body {
				t.Fatal("exclusive creation overwrote existing rules")
			}
		})
	}
}
