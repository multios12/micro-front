package validate

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var categoryPattern = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)
var (
	markdownImagePattern = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	markdownLinkPattern  = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	markdownHeadingMark  = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	markdownListMark     = regexp.MustCompile(`(?m)^\s*(?:[-*+]\s+|\d+\.\s+)`)
	markdownTableMark    = regexp.MustCompile(`\|`)
)

func Length(s string) int {
	return utf8.RuneCountInString(s)
}

func IsCategory(value string) bool {
	return value != "" && categoryPattern.MatchString(value)
}

func IsDateTime(value string) bool {
	_, err := time.Parse("2006-01-02 15:04:05", value)
	return err == nil
}

func SummaryFromContent(content string) string {
	content = markdownImagePattern.ReplaceAllString(content, "$1")
	content = markdownLinkPattern.ReplaceAllString(content, "$1")
	content = markdownHeadingMark.ReplaceAllString(content, "")
	content = markdownListMark.ReplaceAllString(content, "")
	content = markdownTableMark.ReplaceAllString(content, " ")
	cleaned := strings.NewReplacer(
		"\r", " ",
		"\n", " ",
		"*", "",
		"`", "",
		">", "",
		"[", "",
		"]", "",
		"(", "",
		")", "",
		"_", "",
		"~", "",
		"-", " ",
	).Replace(content)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	runes := []rune(cleaned)
	if len(runes) > 140 {
		return string(runes[:140])
	}
	return cleaned
}
