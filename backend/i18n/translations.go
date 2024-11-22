package i18n

import (
	"regexp"
	"strings"
)

type Translations map[string]string
type Replacements map[string]string

func (t Translations) Translate(key string, replacements Replacements) string {
	var val, ok = t[key]
	if !ok || val == "" {
		return ""
	}
	for rkey, rval := range replacements {
		val = strings.ReplaceAll(val, rkey, rval)
	}
	return val
}

// Define a regular expression to match content within {{ }}
var ReMatchBracketContent = regexp.MustCompile(`{{\s*([^{}]+)\s*}}`)

func Translate(tr Translations, key string, reps ...string) string {
	var val, ok = tr[key]
	if !ok || val == "" {
		return ""
	}

	// Find all bracket matches in the input string
	matches := ReMatchBracketContent.FindAllStringSubmatch(val, -1)
	var variables []string
	for _, match := range matches {
		if len(match) > 1 {
			variables = append(variables, match[0])
		}
	}
	var replacements = Replacements{}
	if len(variables) >= len(reps) {
		for idx, v := range variables {
			replacements[v] = reps[idx]
		}
	}
	// fmt.Println("Replacements", key, matches, variables, replacements)

	// translate
	for rkey, rval := range replacements {
		val = strings.ReplaceAll(val, rkey, rval)
	}
	return val
}
