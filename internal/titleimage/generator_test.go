package titleimage

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateSVG_GeneratesAllTemplates(t *testing.T) {
	for _, tmpl := range ListTemplates() {
		svg, err := GenerateSVG(GenerateInput{
			Title:    "ブログタイトル画像ジェネレータ",
			Template: tmpl.ID,
		})
		if err != nil {
			t.Fatalf("GenerateSVG(%s): %v", tmpl.ID, err)
		}
		for _, want := range []string{`<svg xmlns="http://www.w3.org/2000/svg"`, `viewBox="0 0 1200 675"`, `</svg>`} {
			if !strings.Contains(svg, want) {
				t.Fatalf("svg for %s missing %q:\n%s", tmpl.ID, want, svg)
			}
		}
	}
}

func TestGenerateSVG_EscapesTitle(t *testing.T) {
	svg, err := GenerateSVG(GenerateInput{
		Title:    `<script>&"'</script>`,
		Template: TemplateDiary,
	})
	if err != nil {
		t.Fatalf("GenerateSVG: %v", err)
	}
	for _, want := range []string{`&lt;script&gt;`, `&amp;`, `&#34;`, `&#39;`} {
		if !strings.Contains(svg, want) {
			t.Fatalf("escaped svg missing %q:\n%s", want, svg)
		}
	}
	if strings.Contains(svg, `<script>`) || strings.Contains(svg, `"</script>`) {
		t.Fatalf("svg contains unescaped title:\n%s", svg)
	}
}

func TestGenerateSVG_InvalidTemplate(t *testing.T) {
	_, err := GenerateSVG(GenerateInput{
		Title:    "title",
		Template: TemplateID("unknown"),
	})
	if !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("err = %v, want ErrInvalidTemplate", err)
	}
}

func TestGenerateSVG_DefaultsToDiary(t *testing.T) {
	svg, err := GenerateSVG(GenerateInput{Title: "title"})
	if err != nil {
		t.Fatalf("GenerateSVG: %v", err)
	}
	if !strings.Contains(svg, `id="diary-bg"`) {
		t.Fatalf("default template should be diary:\n%s", svg)
	}
}

func TestGenerateSVG_SampleTitleUsesSameFontSize(t *testing.T) {
	for _, tmpl := range ListTemplates() {
		svg, err := GenerateSVG(GenerateInput{
			Title:    "ブログタイトル画像ジェネレータ",
			Template: tmpl.ID,
		})
		if err != nil {
			t.Fatalf("GenerateSVG(%s): %v", tmpl.ID, err)
		}
		if !strings.Contains(svg, `font-weight="700" font-size="84"`) {
			t.Fatalf("title font size for %s should be 84px:\n%s", tmpl.ID, svg)
		}
	}
}

func TestGenerateSVG_TitleIsCenteredHorizontally(t *testing.T) {
	for _, tmpl := range ListTemplates() {
		svg, err := GenerateSVG(GenerateInput{
			Title:    "ブログタイトル画像ジェネレータ",
			Template: tmpl.ID,
		})
		if err != nil {
			t.Fatalf("GenerateSVG(%s): %v", tmpl.ID, err)
		}
		for _, want := range []string{`text-anchor="middle"`, `<text x="600"`} {
			if !strings.Contains(svg, want) {
				t.Fatalf("title for %s should be horizontally centered with %q:\n%s", tmpl.ID, want, svg)
			}
		}
	}
}
