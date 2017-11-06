package util

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/ramin0/submit/config"
)

// FormatTeamName func
func FormatTeamName(team interface{}) string {
	return fmt.Sprintf(config.TeamNameFormat, team)
}

// ParseTeamName func
func ParseTeamName(teamName string) (team int) {
	var teamString string
	fmt.Sscanf(teamName, config.TeamNameFormat, &teamString)
	team, _ = strconv.Atoi(teamString)
	return
}

// TrimTeamName func
func TrimTeamName(team interface{}) string {
	return regexp.MustCompile("[^\\d]").ReplaceAllString(fmt.Sprintf("%s", team), "")
}
