package postgres

import "strings"

var likeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

// ContainsPattern escapes SQL LIKE metacharacters. The query adds the surrounding
// wildcards and must specify ESCAPE '!'.
func ContainsPattern(value string) string {
	return likeEscaper.Replace(value)
}
