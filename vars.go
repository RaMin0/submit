package submit

import (
	"regexp"
	"strings"

	"github.com/ramin0/submit/config"
)

var (
	cookieName  = strings.ToLower(regexp.MustCompile("[^\\w]").ReplaceAllString(config.SubmitName, "-") + "-submit_session-id")
	maxPostSize = int64(50 * 1024 * 1024)

	sessions = map[string]*Session{}
)
