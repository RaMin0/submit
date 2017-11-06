package submit

import (
	"regexp"
	"strings"

	"github.com/ramin0/submit/config"
)

var (
	maxPostSize = int64(50 * 1024 * 1024)

	sessions = map[string]*Session{}
)

func cookieName() string {
	return strings.ToLower(regexp.MustCompile("[^\\w]").ReplaceAllString(config.SubmitName, "-") + "-submit_session-id")
}
