package titleimage

import (
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"strings"
	"unicode"
)

var ErrInvalidTemplate = errors.New("invalid title image template")

//go:embed assets/book-stack.png
var bookStackPNG []byte

//go:embed assets/travel-map.png
var travelMapPNG []byte

const (
	svgWidth  = 1200
	svgHeight = 675
)

var templates = []Template{
	{ID: TemplateTech, Label: "Tech", Description: "無機質・発光。ターミナルをイメージしたもの"},
	{ID: TemplateBook, Label: "Book", Description: "紙・古書"},
	{ID: TemplateDiary, Label: "Diary", Description: "淡い抽象"},
	{ID: TemplateTravel, Label: "Travel", Description: "地図・等高線"},
}

type titleLayout struct {
	X          int
	Y          int
	LineHeight int
	Anchor     string
	Fill       string
	FontFamily string
	MaxUnits   int
}

type categoryLayout struct {
	X          int
	Y          int
	Fill       string
	Opacity    string
	FontFamily string
	MaxUnits   int
}

const titleTextScale = 1.5

func ListTemplates() []Template {
	items := make([]Template, len(templates))
	copy(items, templates)
	return items
}

func IsValidTemplate(id TemplateID) bool {
	for _, tmpl := range templates {
		if tmpl.ID == id {
			return true
		}
	}
	return false
}

func GenerateSVG(input GenerateInput) (string, error) {
	if input.Template == "" {
		input.Template = DefaultTemplate
	}
	if !IsValidTemplate(input.Template) {
		return "", fmt.Errorf("%w: %s", ErrInvalidTemplate, input.Template)
	}

	switch input.Template {
	case TemplateTech:
		return renderTech(input.Title, input.Category), nil
	case TemplateBook:
		return renderBook(input.Title, input.Category), nil
	case TemplateDiary:
		return renderDiary(input.Title, input.Category), nil
	case TemplateTravel:
		return renderTravel(input.Title, input.Category), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTemplate, input.Template)
	}
}

func renderTech(title, category string) string {
	body := `<defs>
<linearGradient id="tech-bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#070b12"/><stop offset="58%" stop-color="#101827"/><stop offset="100%" stop-color="#041416"/></linearGradient>
<pattern id="tech-grid" width="48" height="48" patternUnits="userSpaceOnUse"><path d="M 48 0 L 0 0 0 48" fill="none" stroke="#1ce7ff" stroke-opacity="0.16" stroke-width="1"/></pattern>
<pattern id="tech-dots" width="34" height="34" patternUnits="userSpaceOnUse"><circle cx="3" cy="3" r="1.5" fill="#26efff" opacity="0.42"/></pattern>
<filter id="soft-glow"><feGaussianBlur stdDeviation="6" result="blur"/><feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge></filter>
<filter id="tech-noise"><feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" seed="8"/><feColorMatrix type="saturate" values="0"/><feComponentTransfer><feFuncA type="table" tableValues="0 0.13"/></feComponentTransfer></filter>
</defs>
<rect width="1200" height="675" fill="url(#tech-bg)"/>
<rect width="1200" height="675" fill="url(#tech-grid)"/>
<rect width="1200" height="675" filter="url(#tech-noise)" opacity="0.65"/>
<rect x="84" y="110" width="520" height="334" fill="none" stroke="#25dfff" stroke-opacity="0.16"/>
<rect x="850" y="150" width="244" height="372" fill="none" stroke="#25dfff" stroke-opacity="0.14"/>
<rect x="982" y="44" width="154" height="154" fill="url(#tech-dots)" opacity="0.9"/>
<rect x="330" y="472" width="238" height="106" fill="url(#tech-dots)" opacity="0.52"/>
<g opacity="0.3" stroke="#71ffb5" stroke-width="2"><path d="M104 136h260M104 184h170M104 232h316M824 122h218M860 174h248M786 226h146"/></g>
<g opacity="0.22" stroke="#8af7ff"><path d="M0 96h1200M0 288h1200M0 480h1200M208 0v675M992 0v675"/></g>
<rect x="28" y="34" width="1144" height="607" rx="18" fill="#07101b" fill-opacity="0.44" stroke="#4cecff" stroke-opacity="0.45"/>
<circle cx="1030" cy="143" r="54" fill="#20f7ff" opacity="0.2" filter="url(#soft-glow)"/>
<circle cx="940" cy="514" r="72" fill="#45ff9c" opacity="0.14" filter="url(#soft-glow)"/>
<path d="M612 486h140l64-106h128l52-86h156" fill="none" stroke="#25dfff" stroke-width="3" opacity="0.75"/>
<g fill="#25dfff" stroke="#07101b" stroke-width="4"><circle cx="612" cy="486" r="8"/><circle cx="752" cy="486" r="12"/><circle cx="944" cy="380" r="9"/></g>
<g fill="none" stroke="#63f6ff" stroke-width="5" stroke-linecap="round" stroke-linejoin="round">
<path d="M88 76l20 20-20 20"/>
<path d="M126 116h28"/>
</g>`
	return wrapSVG(body + renderCategory(category, categoryLayout{X: 178, Y: 96, Fill: "#9af8ff", Opacity: "0.88", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 28}) + renderTitle(title, titleLayout{X: svgWidth / 2, Y: 330, LineHeight: 72, Anchor: "middle", Fill: "#f3ffff", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 46}))
}

func renderBook(title, category string) string {
	body := `<defs>
<radialGradient id="book-vignette" cx="50%" cy="44%" r="72%"><stop offset="0%" stop-color="#fff5dc"/><stop offset="68%" stop-color="#ead7b6"/><stop offset="100%" stop-color="#caa879"/></radialGradient>
<pattern id="book-paper" width="56" height="56" patternUnits="userSpaceOnUse"><circle cx="8" cy="10" r="1.1" fill="#8a6b3f" opacity="0.16"/><circle cx="25" cy="21" r="0.9" fill="#31404a" opacity="0.08"/><circle cx="44" cy="35" r="1.4" fill="#9b7749" opacity="0.12"/><path d="M0 18 C14 14 28 22 56 16M0 43 C18 48 34 38 56 44" fill="none" stroke="#7c603d" stroke-opacity="0.08"/></pattern>
<filter id="book-noise"><feTurbulence type="fractalNoise" baseFrequency="0.035 0.22" numOctaves="5" seed="3"/><feColorMatrix type="matrix" values="0.72 0 0 0 0.18 0 0.56 0 0 0.12 0 0 0.34 0 0.04 0 0 0 0.32 0"/></filter>
<filter id="book-fibers"><feTurbulence type="fractalNoise" baseFrequency="0.012 0.85" numOctaves="3" seed="9"/><feColorMatrix type="saturate" values="0"/><feComponentTransfer><feFuncA type="table" tableValues="0 0.18"/></feComponentTransfer></filter>
</defs>
<rect width="1200" height="675" fill="url(#book-vignette)"/>
<rect width="1200" height="675" fill="url(#book-paper)"/>
<rect width="1200" height="675" filter="url(#book-noise)"/>
<rect width="1200" height="675" filter="url(#book-fibers)"/>
<ellipse cx="920" cy="156" rx="230" ry="92" fill="#8c6132" opacity="0.08"/>
<ellipse cx="286" cy="548" rx="270" ry="86" fill="#704b2b" opacity="0.07"/>
<path d="M0 0h1200v675H0z" fill="none" stroke="#76512c" stroke-width="42" stroke-opacity="0.1"/>
<rect x="24" y="30" width="1152" height="615" rx="16" fill="#fff1d0" opacity="0.38" stroke="#6d4e2e" stroke-opacity="0.28"/>
<path d="M154 176h892M154 181h892M154 504h892M154 510h892" stroke="#4d3822" stroke-width="1.5" opacity="0.62"/>
<path d="M184 220h360M184 464h300" stroke="#6d4e2e" stroke-width="1" opacity="0.18"/>` + renderBookStackImage() + `
<g font-family="Georgia, 'Times New Roman', serif" fill="#2f2720">
<path d="M84 76c10-6 21-6 31 0v44c-10-6-21-6-31 0zM119 76c10-6 21-6 31 0v44c-10-6-21-6-31 0z" fill="#2f2720" opacity="0.86"/>
</g>`
	return wrapSVG(body + renderCategory(category, categoryLayout{X: 170, Y: 98, Fill: "#2f2720", Opacity: "0.82", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 28}) + renderTitle(title, titleLayout{X: svgWidth / 2, Y: 330, LineHeight: 78, Anchor: "middle", Fill: "#2f2720", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 46}))
}

func renderBookStackImage() string {
	encoded := base64.StdEncoding.EncodeToString(bookStackPNG)
	return fmt.Sprintf(`<image href="data:image/png;base64,%s" x="732" y="208" width="386" height="276" opacity="0.36" preserveAspectRatio="xMidYMid meet"/>`, encoded)
}

func renderDiary(title, category string) string {
	body := `<defs>
<linearGradient id="diary-bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#d7e1e8"/><stop offset="46%" stop-color="#b9c6c8"/><stop offset="100%" stop-color="#d5dccf"/></linearGradient>
<filter id="diary-blur"><feGaussianBlur stdDeviation="30"/></filter>
<filter id="diary-noise"><feTurbulence type="fractalNoise" baseFrequency="0.58" numOctaves="3" seed="13"/><feColorMatrix type="saturate" values="0"/><feComponentTransfer><feFuncA type="table" tableValues="0 0.16"/></feComponentTransfer></filter>
<pattern id="diary-dots" width="22" height="22" patternUnits="userSpaceOnUse"><circle cx="3" cy="3" r="1.4" fill="#f4f6f2" opacity="0.8"/></pattern>
</defs>
<rect width="1200" height="675" fill="url(#diary-bg)"/>
<rect width="1200" height="675" filter="url(#diary-noise)"/>
<circle cx="250" cy="166" r="170" fill="#6f8fa8" opacity="0.48" filter="url(#diary-blur)"/>
<circle cx="858" cy="210" r="192" fill="#87968b" opacity="0.42" filter="url(#diary-blur)"/>
<circle cx="690" cy="532" r="184" fill="#7b8f8c" opacity="0.4" filter="url(#diary-blur)"/>
<circle cx="1070" cy="104" r="178" fill="none" stroke="#eef2ec" stroke-width="2" opacity="0.58"/>
<circle cx="842" cy="676" r="230" fill="#5f788d" opacity="0.24"/>
<rect x="996" y="72" width="110" height="110" fill="url(#diary-dots)" opacity="0.74"/>
<rect x="84" y="468" width="156" height="88" fill="url(#diary-dots)" opacity="0.42"/>
<rect x="934" y="492" width="150" height="98" fill="url(#diary-dots)" opacity="0.48"/>
<path d="M140 470 C260 386 332 548 464 450 S694 354 822 438 1014 480 1080 390" fill="none" stroke="#314655" stroke-width="3" opacity="0.2"/>
<path d="M168 180 C308 130 392 212 520 174 S760 94 928 158" fill="none" stroke="#44574d" stroke-width="2" opacity="0.18"/>
<rect x="24" y="30" width="1152" height="615" rx="16" fill="#f5f6ef" fill-opacity="0.16" stroke="#eef2ec" stroke-opacity="0.65"/>
<g font-family="system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif" fill="#5c5268">
<path d="M84 116l30-30 18 18-30 30-27 8zM118 82l8-8c4-4 11-4 15 0l4 4c4 4 4 11 0 15l-8 8z" fill="#34404a" opacity="0.9"/>
</g>`
	return wrapSVG(body + renderCategory(category, categoryLayout{X: 158, Y: 108, Fill: "#202b33", Opacity: "0.78", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 28}) + renderTitle(title, titleLayout{X: svgWidth / 2, Y: 330, LineHeight: 76, Anchor: "middle", Fill: "#202b33", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 46}))
}

func renderTravel(title, category string) string {
	body := `<defs>
<linearGradient id="travel-title-wash" x1="0" y1="0" x2="1" y2="0"><stop offset="0%" stop-color="#f4efd9" stop-opacity="0.74"/><stop offset="58%" stop-color="#f4efd9" stop-opacity="0.36"/><stop offset="100%" stop-color="#f4efd9" stop-opacity="0"/></linearGradient>
</defs>
` + renderTravelMapImage() + `
<rect width="1200" height="675" fill="url(#travel-title-wash)"/>
<rect x="24" y="30" width="1152" height="615" rx="16" fill="none" stroke="#5f746a" stroke-opacity="0.34"/>
<g font-family="system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif" fill="#334d35">
<path d="M104 74c-15 0-27 12-27 27 0 22 27 51 27 51s27-29 27-51c0-15-12-27-27-27zm0 37c-7 0-12-5-12-12s5-12 12-12 12 5 12 12-5 12-12 12z" fill="#334d35"/>
<text x="84" y="178" font-size="20" opacity="0.78">35.6895N 139.6917E</text>
</g>`
	return wrapSVG(body + renderCategory(category, categoryLayout{X: 150, Y: 113, Fill: "#26332f", Opacity: "0.84", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 28}) + renderTitle(title, titleLayout{X: svgWidth / 2, Y: 330, LineHeight: 74, Anchor: "middle", Fill: "#26332f", FontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif", MaxUnits: 46}))
}

func renderTravelMapImage() string {
	encoded := base64.StdEncoding.EncodeToString(travelMapPNG)
	return fmt.Sprintf(`<image href="data:image/png;base64,%s" x="0" y="0" width="1200" height="675" preserveAspectRatio="xMidYMid slice"/>`, encoded)
}

func wrapSVG(body string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">%s</svg>`, svgWidth, svgHeight, svgWidth, svgHeight, body)
}

func renderCategory(category string, layout categoryLayout) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return ""
	}
	category = ellipsizeText(category, layout.MaxUnits)
	return fmt.Sprintf(`<text x="%d" y="%d" font-family="%s" font-size="60" font-weight="700" text-anchor="start" dominant-baseline="middle" fill="%s" opacity="%s">%s</text>`, layout.X, layout.Y, layout.FontFamily, layout.Fill, layout.Opacity, html.EscapeString(category))
}

func renderTitle(title string, layout titleLayout) string {
	lines, fontSize := fitTitle(title, scaledTitleUnits(layout.MaxUnits))
	lineHeight := scaledTitleSize(layout.LineHeight)
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<g font-family="%s" font-weight="700" font-size="%d" text-anchor="%s" fill="%s">`, layout.FontFamily, fontSize, layout.Anchor, layout.Fill))
	startY := layout.Y - ((len(lines) - 1) * lineHeight / 2)
	for i, line := range lines {
		b.WriteString(fmt.Sprintf(`<text x="%d" y="%d">%s</text>`, layout.X, startY+(i*lineHeight), html.EscapeString(line)))
	}
	b.WriteString(`</g>`)
	return b.String()
}

func scaledTitleSize(size int) int {
	return int(float64(size)*titleTextScale + 0.5)
}

func scaledTitleUnits(units int) int {
	scaled := int(float64(units) / titleTextScale)
	if scaled < 1 {
		return 1
	}
	return scaled
}

func fitTitle(title string, maxUnits int) ([]string, int) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled"
	}
	steps := []struct {
		size  int
		units int
	}{
		{scaledTitleSize(56), maxUnits},
		{scaledTitleSize(50), maxUnits + scaledTitleUnits(4)},
		{scaledTitleSize(44), maxUnits + scaledTitleUnits(8)},
		{scaledTitleSize(38), maxUnits + scaledTitleUnits(12)},
	}
	for _, step := range steps {
		lines := wrapRunes(title, step.units, 3)
		if len(lines) <= 3 && strings.Join(lines, "") == strings.Join(wrapRunes(title, step.units, 99), "") {
			return lines, step.size
		}
	}
	minUnits := maxUnits + scaledTitleUnits(12)
	return ellipsizeLines(wrapRunes(title, minUnits, 3), minUnits), scaledTitleSize(38)
}

func wrapRunes(text string, maxUnits, maxLines int) []string {
	var lines []string
	var line []rune
	lineUnits := 0
	for _, r := range []rune(text) {
		units := runeUnits(r)
		if unicode.IsSpace(r) {
			units = 1
		}
		if len(line) > 0 && lineUnits+units > maxUnits {
			lines = append(lines, strings.TrimSpace(string(line)))
			if len(lines) == maxLines {
				return lines
			}
			line = line[:0]
			lineUnits = 0
		}
		line = append(line, r)
		lineUnits += units
	}
	if len(line) > 0 && len(lines) < maxLines {
		lines = append(lines, strings.TrimSpace(string(line)))
	}
	return lines
}

func ellipsizeLines(lines []string, maxUnits int) []string {
	if len(lines) == 0 {
		return []string{"..."}
	}
	last := []rune(strings.TrimSpace(lines[len(lines)-1]))
	for displayUnits(string(last))+3 > maxUnits && len(last) > 0 {
		last = last[:len(last)-1]
	}
	lines[len(lines)-1] = strings.TrimSpace(string(last)) + "..."
	return lines
}

func displayUnits(text string) int {
	units := 0
	for _, r := range text {
		units += runeUnits(r)
	}
	return units
}

func ellipsizeText(text string, maxUnits int) string {
	if displayUnits(text) <= maxUnits {
		return text
	}
	runes := []rune(strings.TrimSpace(text))
	for displayUnits(string(runes))+3 > maxUnits && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimSpace(string(runes)) + "..."
}

func runeUnits(r rune) int {
	if r <= 0x007f {
		return 1
	}
	return 2
}
