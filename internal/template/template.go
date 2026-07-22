package template

import (
	"fmt"
	"regexp"
	"strings"
)

var variablePattern = regexp.MustCompile(
	`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`,
)

func Render(command string, values map[string]string) (string, error) {
	var unresolved []string

	rendered := variablePattern.ReplaceAllStringFunc(
		command,
		func(match string) string {
			groups := variablePattern.FindStringSubmatch(match)
			name := groups[1]

			value, exists := values[name]
			if !exists {
				unresolved = append(unresolved, name)
				return match
			}

			return value
		},
	)

	if len(unresolved) > 0 {
		return "", fmt.Errorf(
			"unresolved variables: %s",
			strings.Join(unique(unresolved), ", "),
		)
	}

	return rendered, nil
}

func Preview(command string, values map[string]string) string {
	return variablePattern.ReplaceAllStringFunc(
		command,
		func(match string) string {
			groups := variablePattern.FindStringSubmatch(match)
			name := groups[1]

			if value, exists := values[name]; exists {
				return value
			}

			return "<" + name + ">"
		},
	)
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
