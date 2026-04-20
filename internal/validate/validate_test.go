package validate

import (
	"strings"
	"testing"
)

func TestSummaryFromContent_RemovesMarkdownSyntax(t *testing.T) {
	got := SummaryFromContent(strings.Join([]string{
		"# 見出し",
		"",
		"![画像|50%](/admin/images/1/1.png)",
		"",
		"本文は **太字** と [リンク](https://example.com) を含みます。",
		"",
		"| 名前 | 説明 |",
		"| --- | --- |",
	}, "\n"))

	for _, disallowed := range []string{"#", "![", "](", "**", "https://", "|", "`"} {
		if strings.Contains(got, disallowed) {
			t.Fatalf("summary contains markdown syntax %q: %q", disallowed, got)
		}
	}
	for _, want := range []string{"見出し", "画像", "50%", "本文は", "太字", "リンク"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q: %q", want, got)
		}
	}
}
