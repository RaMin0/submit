package util

import (
	"fmt"
	"regexp"

	"github.com/ramin0/submit/config"
)

// FormatGroup func
func FormatGroup(group interface{}) string {
	return fmt.Sprintf(config.GroupFormat, group)
}

// FormatTeamName func
func FormatTeamName(team interface{}) string {
	return fmt.Sprintf(config.TeamNameFormat, team)
}

// ParseTeamName func
func ParseTeamName(teamName string) (team int) {
	fmt.Sscanf(teamName, config.TeamNameFormat, &team)
	return
}

// TrimTeamName func
func TrimTeamName(team interface{}) string {
	return regexp.MustCompile("[^\\d]").ReplaceAllString(fmt.Sprintf("%s", team), "")
}
