package markdown

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	orderedListPattern    = regexp.MustCompile(`^\d+\.\s+(.*)$`)
	unorderedListPattern  = regexp.MustCompile(`^-\s+(.*)$`)
	headingPattern        = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	horizontalRulePattern = regexp.MustCompile(`^(?:-{3,}|\*{3,}|_{3,})$`)
	codePattern           = regexp.MustCompile("`([^`]+)`")
	imagePattern          = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+&quot;([^&]*)&quot;)?\)`)
	imageSizePattern      = regexp.MustCompile(`^(\d+%?)(?:x(\d+))?$`)
	linkPattern           = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+&quot;([^&]*)&quot;)?\)`)
	strongPattern         = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	underlinePattern      = regexp.MustCompile(`__([^_]+)__`)
	strikePattern         = regexp.MustCompile(`~~([^~]+)~~`)
	emphasisPattern       = regexp.MustCompile(`\*([^*]+)\*`)
)

const carryOverMarker = "----ここまで前回内容で置換"

// 行単位の Markdown を変換する際の読み取り位置を保持する。
type parser struct {
	lines []string
	pos   int
}

// Markdown の一部記法を HTML に変換する。
// 対応する主な記法は、見出し、番号付き/箇条書きリスト、ネストしたリスト、
// コードブロック、引用、水平線、テーブル、段落、太字、斜体、下線、
// 取り消し線、リンク、インラインコード、画像。
func ToHTML(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	input = strings.Join(filterHiddenLines(strings.Split(input, "\n")), "\n")

	p := parser{
		lines: strings.Split(input, "\n"),
	}

	return strings.Join(p.parseBlocks(0), "\n")
}

func filterHiddenLines(lines []string) []string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == carryOverMarker {
			filtered = append(filtered, "---")
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

// インデントが minIndent 未満になるまでブロック要素を読み取る。
func (p *parser) parseBlocks(minIndent int) []string {
	var blocks []string

	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)
		indent := leadingIndentWidth(line)

		if trimmed == "" {
			p.pos++
			continue
		}
		if indent < minIndent {
			break
		}

		content := trimIndent(line, minIndent)
		switch {
		case strings.HasPrefix(strings.TrimSpace(content), "```"):
			blocks = append(blocks, p.parseCodeBlock(minIndent))
		case isTableStart(p.lines, p.pos, minIndent):
			blocks = append(blocks, p.parseTable(minIndent))
		case isHorizontalRule(content):
			blocks = append(blocks, p.parseHorizontalRule())
		case headingPattern.MatchString(strings.TrimSpace(content)):
			blocks = append(blocks, p.parseHeading(content))
		case strings.HasPrefix(strings.TrimSpace(content), ">"):
			blocks = append(blocks, p.parseBlockquote(minIndent))
		case hasListMarker(content):
			blocks = append(blocks, p.parseList(indent))
		default:
			blocks = append(blocks, p.parseParagraph(minIndent))
		}
	}

	return blocks
}

// Markdown の水平線を hr 要素に変換する。
func (p *parser) parseHorizontalRule() string {
	p.pos++
	return "<hr>"
}

// 閉じフェンスが現れるまでコードをそのまま保持する。
func (p *parser) parseCodeBlock(minIndent int) string {
	p.pos++
	var codeLines []string

	for p.pos < len(p.lines) {
		line := trimIndent(p.lines[p.pos], minIndent)
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			p.pos++
			break
		}
		codeLines = append(codeLines, line)
		p.pos++
	}

	return fmt.Sprintf("<pre><code>%s\n</code></pre>", html.EscapeString(strings.Join(codeLines, "\n")))
}

// # から ###### までの見出しを変換する。
func (p *parser) parseHeading(line string) string {
	matches := headingPattern.FindStringSubmatch(strings.TrimSpace(line))
	p.pos++
	level := len(matches[1])
	return fmt.Sprintf("<h%d>%s</h%d>", level, renderInline(matches[2]), level)
}

// 連続する引用行を 1 つの blockquote にまとめる。
// 各引用行の間には <br> を入れて改行を保持する。
func (p *parser) parseBlockquote(minIndent int) string {
	var quoteLines []string

	for p.pos < len(p.lines) {
		line := trimIndent(p.lines[p.pos], minIndent)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, ">") {
			break
		}
		quoteLines = append(quoteLines, strings.TrimSpace(strings.TrimPrefix(trimmed, ">")))
		p.pos++
	}

	return fmt.Sprintf("<blockquote>\n<p>%s</p>\n</blockquote>", strings.Join(renderInlineLines(quoteLines), "<br>"))
}

// 次のブロック要素が始まるまでの行を 1 つの段落にまとめる。
func (p *parser) parseParagraph(minIndent int) string {
	var lines []string

	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)
		indent := leadingIndentWidth(line)

		if trimmed == "" {
			p.pos++
			break
		}
		if indent < minIndent {
			break
		}

		content := trimIndent(line, minIndent)
		if len(lines) > 0 && startsBlock(content, p.lines, p.pos, minIndent) {
			break
		}

		lines = append(lines, strings.TrimSpace(content))
		p.pos++
	}

	return fmt.Sprintf("<p>%s</p>", renderInline(strings.Join(lines, " ")))
}

// 通常のリストとインデントされたネストリストを再帰的に処理する。
func (p *parser) parseList(indent int) string {
	listLine := trimIndent(p.lines[p.pos], indent)
	listType, _, ok := parseListMarker(listLine)
	if !ok {
		return ""
	}

	type listItem struct {
		text     []string
		children []string
	}

	var items []listItem
	var current *listItem

	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)
		lineIndent := leadingIndentWidth(line)

		if trimmed == "" {
			p.pos++
			continue
		}
		if lineIndent < indent {
			break
		}

		content := trimIndent(line, indent)
		if lineIndent == indent {
			itemType, body, ok := parseListMarker(content)
			if !ok || itemType != listType {
				break
			}
			items = append(items, listItem{text: []string{body}})
			current = &items[len(items)-1]
			p.pos++
			continue
		}

		if current == nil {
			break
		}

		nestedContent := trimIndent(line, lineIndent)
		if hasListMarker(nestedContent) {
			current.children = append(current.children, p.parseList(lineIndent))
			continue
		}

		current.text = append(current.text, strings.TrimSpace(trimIndent(line, indent)))
		p.pos++
	}

	var b strings.Builder
	if listType == "ol" {
		b.WriteString("<ol>\n")
	} else {
		b.WriteString("<ul>\n")
	}

	for _, item := range items {
		body := renderInline(strings.Join(item.text, " "))
		if len(item.children) == 0 {
			b.WriteString(fmt.Sprintf("<li>%s</li>\n", body))
			continue
		}

		b.WriteString("<li>")
		b.WriteString(body)
		b.WriteString("\n")
		b.WriteString(strings.Join(item.children, "\n"))
		b.WriteString("</li>\n")
	}

	if listType == "ol" {
		b.WriteString("</ol>")
	} else {
		b.WriteString("</ul>")
	}

	return b.String()
}

// ヘッダー区切り行を持つパイプ形式テーブルを HTML に変換する。
func (p *parser) parseTable(minIndent int) string {
	header := splitTableRow(trimIndent(p.lines[p.pos], minIndent))
	p.pos += 2

	var bodyRows [][]string
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)
		indent := leadingIndentWidth(line)
		if trimmed == "" || indent < minIndent || !isTableRow(trimIndent(line, minIndent)) {
			break
		}
		bodyRows = append(bodyRows, splitTableRow(trimIndent(line, minIndent)))
		p.pos++
	}

	var b strings.Builder
	b.WriteString("<table>\n<thead>\n<tr>")
	for _, cell := range header {
		b.WriteString(fmt.Sprintf("<th>%s</th>", renderInline(cell)))
	}
	b.WriteString("</tr>\n</thead>")

	if len(bodyRows) > 0 {
		b.WriteString("\n<tbody>")
		for _, row := range bodyRows {
			b.WriteString("\n<tr>")
			for _, cell := range row {
				b.WriteString(fmt.Sprintf("<td>%s</td>", renderInline(cell)))
			}
			b.WriteString("</tr>")
		}
		b.WriteString("\n</tbody>")
	}

	b.WriteString("\n</table>")
	return b.String()
}

// 文字列をエスケープしたあと、インライン Markdown を適用する。
func renderInline(input string) string {
	escaped := html.EscapeString(input)

	type placeholder struct {
		token string
		html  string
	}
	var placeholders []placeholder

	escaped = codePattern.ReplaceAllStringFunc(escaped, func(match string) string {
		subMatches := codePattern.FindStringSubmatch(match)
		token := fmt.Sprintf("__CODE_PLACEHOLDER_%d__", len(placeholders))
		placeholders = append(placeholders, placeholder{
			token: token,
			html:  fmt.Sprintf("<code>%s</code>", subMatches[1]),
		})
		return token
	})

	escaped = imagePattern.ReplaceAllStringFunc(escaped, renderImageHTML)
	escaped = linkPattern.ReplaceAllString(escaped, `<a href="$2">$1</a>`)
	escaped = strongPattern.ReplaceAllString(escaped, `<strong>$1</strong>`)
	escaped = underlinePattern.ReplaceAllString(escaped, `<u>$1</u>`)
	escaped = strikePattern.ReplaceAllString(escaped, `<del>$1</del>`)
	escaped = emphasisPattern.ReplaceAllString(escaped, `<em>$1</em>`)

	for _, item := range placeholders {
		escaped = strings.ReplaceAll(escaped, item.token, item.html)
	}

	return escaped
}

// 各行に対して個別にインライン変換を適用する。
func renderInlineLines(lines []string) []string {
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		rendered = append(rendered, renderInline(line))
	}
	return rendered
}

func renderImageHTML(match string) string {
	subMatches := imagePattern.FindStringSubmatch(match)
	if len(subMatches) < 3 {
		return match
	}

	alt := subMatches[1]
	src := subMatches[2]
	sizeSpec := ""

	if pipe := strings.LastIndex(alt, "|"); pipe >= 0 {
		candidate := strings.TrimSpace(alt[pipe+1:])
		if imageSizePattern.MatchString(candidate) {
			alt = alt[:pipe]
			sizeSpec = candidate
		}
	}

	var b strings.Builder
	b.WriteString(`<img src="`)
	b.WriteString(src)
	b.WriteString(`" alt="`)
	b.WriteString(alt)
	b.WriteString(`"`)

	if sizeSpec != "" {
		sizeMatches := imageSizePattern.FindStringSubmatch(sizeSpec)
		if len(sizeMatches) >= 2 && sizeMatches[1] != "" {
			if strings.HasSuffix(sizeMatches[1], "%") {
				b.WriteString(` style="width: `)
				b.WriteString(sizeMatches[1])
				b.WriteString(`;"`)
			} else {
				b.WriteString(` width="`)
				b.WriteString(sizeMatches[1])
				b.WriteString(`"`)
			}
		}
		if len(sizeMatches) >= 3 && sizeMatches[2] != "" {
			b.WriteString(` height="`)
			b.WriteString(sizeMatches[2])
			b.WriteString(`"`)
		}
	}

	b.WriteString(`>`)
	return b.String()
}

// 現在行が新しいブロック要素の開始かどうかを判定する。
func startsBlock(content string, lines []string, pos int, minIndent int) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "```") ||
		isHorizontalRule(content) ||
		headingPattern.MatchString(trimmed) ||
		strings.HasPrefix(trimmed, ">") ||
		hasListMarker(content) ||
		isTableStart(lines, pos, minIndent)
}

// 行が Markdown の水平線かどうかを判定する。
func isHorizontalRule(line string) bool {
	line = strings.ReplaceAll(strings.TrimSpace(line), " ", "")
	return horizontalRulePattern.MatchString(line)
}

// 行頭に番号付きまたは箇条書きリスト記号があるかを判定する。
func hasListMarker(line string) bool {
	_, _, ok := parseListMarker(strings.TrimSpace(line))
	return ok
}

// 1 行のリスト項目からリスト種別と本文を取り出す。
func parseListMarker(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if matches := orderedListPattern.FindStringSubmatch(line); matches != nil {
		return "ol", matches[1], true
	}
	if matches := unorderedListPattern.FindStringSubmatch(line); matches != nil {
		return "ul", matches[1], true
	}
	return "", "", false
}

// 現在行と次行が Markdown テーブルの開始かどうかを判定する。
func isTableStart(lines []string, pos int, minIndent int) bool {
	if pos+1 >= len(lines) {
		return false
	}
	return isTableRow(trimIndent(lines[pos], minIndent)) &&
		isTableSeparator(trimIndent(lines[pos+1], minIndent))
}

// 行がテーブル行として扱える最小限のパイプ構造を持つかを判定する。
func isTableRow(line string) bool {
	line = strings.TrimSpace(line)
	return strings.Count(line, "|") >= 2
}

// isTableSeparator は行が Markdown テーブルの区切り行かどうかを判定する。
func isTableSeparator(line string) bool {
	line = strings.TrimSpace(line)
	if !isTableRow(line) {
		return false
	}

	for _, cell := range splitTableRow(line) {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		for _, r := range cell {
			if r != '-' && r != ':' {
				return false
			}
		}
	}

	return true
}

// splitTableRow は Markdown のテーブル行をセルごとの文字列へ分割する。
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// leadingIndentWidth はブロック解析用に先頭の空白幅を数える。
func leadingIndentWidth(line string) int {
	width := 0
	for _, r := range line {
		if r == ' ' {
			width++
			continue
		}
		if r == '\t' {
			width += 2
			continue
		}
		break
	}
	return width
}

// trimIndent は行頭から最大 width 分のインデントを取り除く。
func trimIndent(line string, width int) string {
	for width > 0 && len(line) > 0 {
		switch line[0] {
		case ' ':
			line = line[1:]
			width--
		case '\t':
			line = line[1:]
			width -= 2
		default:
			return line
		}
	}
	return line
}
