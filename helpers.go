package submit

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ramin0/submit/config"
	httpntlm "github.com/vadimi/go-http-ntlm"
	calendar "google.golang.org/api/calendar/v3"
)

var (
	studentApplicationNoRegexp = regexp.MustCompile("^\\d{1,}-\\d{4,5}$")
)

func render(w io.Writer, r *http.Request, t string, data interface{}) {
	tmplDir := path.Join(rootPath(), "templates")
	tmpl := template.New(tmplDir)
	tmpl = tmpl.Funcs(template.FuncMap{
		"md5": func() string {
			return time.Now().Format("20060102150405")
		},
		"activeNav": func(path string) bool {
			return r.URL.Path == path
		},
		"currentURL": func() *url.URL {
			return r.URL
		},
		"loggedIn": func() bool {
			return isLoggedIn(r)
		},
		"currentUser": func() *User {
			return currentUser(r)
		},
		"empty": func(s interface{}) bool {
			return s == nil || s.(string) == ""
		},
		"simpleFormat": func(s interface{}) template.HTML {
			if s == nil {
				return ""
			}

			html := strings.Replace(s.(string), "\n", "<br />", -1)
			return template.HTML(html)
		},
		"raw": func(s interface{}) template.HTML {
			if s == nil {
				return ""
			}

			return template.HTML(s.(string))
		},
		"params": func(s interface{}) string {
			return r.URL.Query().Get(s.(string))
		},
		"string": func(s interface{}) string {
			return fmt.Sprintf("%v", s)
		},
		"now": func() time.Time {
			return time.Now()
		},
		"submitName": func() string {
			return config.SubmitName
		},
		"feature": func(name string) bool {
			return featureEnabled(name)
		},
	})
	tmpl = template.Must(tmpl.ParseFiles(fmt.Sprintf(path.Join(tmplDir, "%s.tmpl"), t)))
	tmpl = template.Must(tmpl.ParseGlob(path.Join(tmplDir, "layouts", "*.tmpl")))

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		panic(err)
	}
}

func renderJSON(w http.ResponseWriter, data interface{}, status ...int) {
	w.Header().Add("Content-Type", "application/json; charset=utf8")
	if len(status) > 0 {
		w.WriteHeader(status[0])
	}
	json.NewEncoder(w).Encode(data)
}

func featureEnabled(name string) bool {
	return config.FeaturesEnabled[name]
}

func currentSession(r *http.Request) *Session {
	sessionCookie, err := r.Cookie(cookieName())
	if err != nil {
		return nil
	}

	sessionID := sessionCookie.Value
	session, sessionOk := sessions[sessionID]
	if !sessionOk {
		return nil
	}

	return session
}

func currentUser(r *http.Request) *User {
	session := currentSession(r)
	if session != nil {
		return session.User
	}
	return nil
}

func isLoggedIn(r *http.Request) bool {
	return currentUser(r) != nil
}

func logIn(username, password string) (*User, error) {
	if strings.HasPrefix(username, "admin:") && password == config.AdminPassword {
		username = strings.TrimPrefix(username, "admin:")
		return &User{
			ID:        username,
			FullName:  fmt.Sprintf("Administrator (%s)", username),
			UserName:  username,
			group:     "admins",
			teamName:  "Administrators",
			teamGroup: "admins",
		}, nil
	}

	ntlmServer := "http://student.guc.edu.eg"
	ntlmPath := "/External/Student/Data/UpdateSystemUserData.aspx"

	client := http.Client{
		Transport: &httpntlm.NtlmTransport{
			Domain:   "GUC",
			User:     username,
			Password: password,
		},
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", ntlmServer, ntlmPath), nil)
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("Invalid username or password")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to retrieve student data")
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	studentTable := doc.Find("#Table2").First()
	studentApplicationNo := studentTable.Find("#L_StudentApplicationNo").Text()
	studentFullName := studentTable.Find("#L_StudentFullName").Text()

	if !studentApplicationNoRegexp.MatchString(studentApplicationNo) ||
		strings.TrimSpace(studentFullName) == "" {
		return nil, fmt.Errorf("Failed to retrieve student data")
	}

	return &User{
		ID:       studentApplicationNo,
		FullName: studentFullName,
		UserName: username,
	}, nil
}

func persistUser(w http.ResponseWriter, user *User) {
	if user == nil {
		return
	}

	var sessionID string
	for {
		hasher := md5.New()
		hasher.Write([]byte(strconv.FormatInt(time.Now().Unix()*rand.Int63(), 10)))
		sessionID = hex.EncodeToString(hasher.Sum(nil))

		if _, ok := sessions[sessionID]; !ok {
			break
		}
	}

	sessionTimestamp := time.Now()
	if cairo, err := time.LoadLocation("Africa/Cairo"); err == nil {
		sessionTimestamp = sessionTimestamp.In(cairo)
	}

	sessions[sessionID] = &Session{
		Timestamp: sessionTimestamp,
		History:   []string{},
		User:      user,
	}

	http.SetCookie(w, &http.Cookie{
		Name:    cookieName(),
		Value:   sessionID,
		Expires: time.Now().Add(365 * 24 * time.Hour),
	})
}

func unpersistUser(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie(cookieName())
	if err == nil {
		sessionID := sessionCookie.Value
		delete(sessions, sessionID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    cookieName(),
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
	})
}

func ensureLoggedIn(w http.ResponseWriter, r *http.Request) bool {
	if !isLoggedIn(r) {
		loginURL, err := url.Parse("/login")
		if err != nil {
			return false
		}

		loginURLQuery := loginURL.Query()
		loginURLQuery.Add("u", r.URL.Path)
		loginURL.RawQuery = loginURLQuery.Encode()
		http.Redirect(w, r, loginURL.String(), http.StatusFound)

		return false
	}

	return true
}

func ensureLoggedInAdmin(w http.ResponseWriter, r *http.Request) bool {
	return ensureLoggedIn(w, r) && currentUser(r).Admin()
}

func rootPath() string {
	if path, ok := os.LookupEnv("SUBMIT_ROOT_PATH"); ok {
		return path
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		return path.Dir(file)
	}

	return ""
}

func newSlotFromEvent(event *calendar.Event) *Slot {
	eventDateTime, _ := time.Parse(time.RFC3339, event.Start.DateTime)
	eventDate := eventDateTime.Format("Monday, January 2")
	eventTime := eventDateTime.Format("3:04 PM")

	return &Slot{
		ID:   event.Id,
		Date: eventDate,
		Time: eventTime,
	}
}
