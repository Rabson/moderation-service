package moderation

import (
	"regexp"
	"strings"
)

var (
	nonAlnumSpace = regexp.MustCompile(`[^a-z0-9\s]`)
	multiSpace    = regexp.MustCompile(`\s+`)
)

var leetReplacer = strings.NewReplacer(
	"@", "a",
	"4", "a",
	"3", "e",
	"1", "i",
	"!", "i",
	"0", "o",
	"5", "s",
	"$", "s",
	"7", "t",
)

func Preprocess(input string) string {
	text := strings.ToLower(strings.TrimSpace(input))
	text = leetReplacer.Replace(text)
	text = nonAlnumSpace.ReplaceAllString(text, " ")
	text = multiSpace.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
