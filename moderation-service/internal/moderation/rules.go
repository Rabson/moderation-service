package moderation

import "regexp"

type RuleEngine struct {
	hatePatterns     []*regexp.Regexp
	violencePatterns []*regexp.Regexp
	sexualPatterns   []*regexp.Regexp
	spamPatterns     []*regexp.Regexp
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		hatePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b(hate|racist|bigot|nazi|slur)\b`),
			regexp.MustCompile(`\b(kill\s+all\s+\w+)\b`),
		},
		violencePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b(kill|murder|shoot|stab|bomb|assault)\b`),
			regexp.MustCompile(`\b(i\s+will\s+hurt\s+you)\b`),
		},
		sexualPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b(sex|nude|porn|explicit|xxx)\b`),
			regexp.MustCompile(`\b(send\s+nudes?)\b`),
		},
		spamPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b(free\s+money|click\s+here|buy\s+now|guaranteed\s+profit)\b`),
			regexp.MustCompile(`https?://`),
		},
	}
}

func (r *RuleEngine) Score(text string) Labels {
	labels := Labels{}

	if matchesAny(r.hatePatterns, text) {
		labels.Hate = 0.9
	}
	if matchesAny(r.violencePatterns, text) {
		labels.Violence = 0.9
	}
	if matchesAny(r.sexualPatterns, text) {
		labels.Sexual = 0.85
	}
	if matchesAny(r.spamPatterns, text) {
		labels.Spam = 0.8
	}

	maxBad := max4(labels.Hate, labels.Violence, labels.Sexual, labels.Spam)
	labels.Safe = clamp01(1 - maxBad)
	return labels
}

func matchesAny(patterns []*regexp.Regexp, text string) bool {
	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}
