package submit

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/ramin0/submit/config"
	"github.com/ramin0/submit/lib/google"
	"github.com/ramin0/submit/lib/slack"
)

var (
	panicHandler = func(r *http.Request, err interface{}) { panic(err) }
)

// Mux func
func Mux() http.Handler {
	m := http.NewServeMux()

	for _, f := range []func() (string, http.HandlerFunc){
		root, webhook,
		login, logout,
		proposal, submit, evaluation,
		settings, settingsSlack,
		adminSessions,
	} {
		pattern, fn := f()

		for _, mw := range []func(http.HandlerFunc) http.HandlerFunc{
			wrap,
			sessionLog,
		} {
			fn = mw(fn)
		}

		m.HandleFunc(pattern, wrap(fn))
	}

	publicDir := path.Join(rootPath(), "public")
	public := http.FileServer(http.Dir(publicDir))
	m.Handle("/stylesheets/", public)
	m.Handle("/javascripts/", public)
	m.Handle("/images/", public)
	m.Handle("/fonts/", public)
	m.Handle("/favicon.ico", public)

	return m
}

// OnPanic func
func OnPanic(fn func(r *http.Request, err interface{})) {
	panicHandler = fn
}

func wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				panicHandler(r, errors.Wrap(err, 1))
			}
		}()

		next(w, r)
	}
}

func sessionLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s := currentSession(r); s != nil {
			if len(s.History) == 5 {
				s.History = s.History[0:4]
			}
			s.History = append([]string{r.URL.Path}, s.History...)
		}

		next(w, r)
	}
}

func root() (string, http.HandlerFunc) {
	return "/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if !ensureLoggedIn(w, r) {
			return
		}

		render(w, r, "home", nil)
	}
}

func login() (string, http.HandlerFunc) {
	return "/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.ParseForm()

			username := strings.TrimSpace(r.FormValue("session[username]"))
			password := strings.TrimSpace(r.FormValue("session[password]"))

			username = strings.Split(username, "@")[0]

			if username == "" || password == "" {
				render(w, r, "login", map[string]string{
					"Flash":    "Make sure all fields are populated",
					"Username": username,
				})
				return
			}

			user, err := logIn(username, password)
			if err != nil {
				render(w, r, "login", map[string]string{
					"Flash":    err.Error(),
					"Username": username,
				})
				return
			}

			persistUser(w, user)

			if u := r.URL.Query().Get("u"); u != "" {
				http.Redirect(w, r, u, http.StatusFound)
			} else {
				http.Redirect(w, r, "/", http.StatusFound)
			}
		} else {
			render(w, r, "login", nil)
		}
	}
}

func logout() (string, http.HandlerFunc) {
	return "/logout", func(w http.ResponseWriter, r *http.Request) {
		unpersistUser(w, r)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func proposal() (string, http.HandlerFunc) {
	return "/proposal", func(w http.ResponseWriter, r *http.Request) {
		if !featureEnabled("proposals") {
			http.NotFound(w, r)
			return
		}

		if !ensureLoggedIn(w, r) {
			return
		}

		render(w, r, "proposal", nil)
	}
}

func submit() (string, http.HandlerFunc) {
	return "/submit", func(w http.ResponseWriter, r *http.Request) {
		if !featureEnabled("submissions") {
			http.NotFound(w, r)
			return
		}

		if !ensureLoggedIn(w, r) {
			return
		}

		deadline, _ := time.Parse(time.RFC3339, config.SubmissionDeadline)
		if time.Now().After(deadline) {
			render(w, r, "submit", map[string]bool{"DeadlinePassed": true})
			return
		}

		if featureEnabled("evaluations") {
			if slot, _ := google.CalendarTeamSlot(currentUser(r).TeamName()); slot == nil {
				render(w, r, "submit", map[string]bool{"EvaluationMissing": true})
				return
			}
		}

		if r.Method == http.MethodPost {
			var err error
			if err = r.ParseMultipartForm(maxPostSize); nil != err {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			data := map[string]interface{}{}

			for _, item := range config.SubmissionsItems {
				t := item["Type"]
				data[t] = nil

				switch t {
				case "url":
					if url := strings.TrimSpace(r.FormValue("submission[url]")); len(url) > 0 {
						data[t] = url
					}
				case "file":
					if fileHeaders := r.MultipartForm.File["submission[file]"]; len(fileHeaders) > 0 {
						if file, err := fileHeaders[0].Open(); err == nil {
							data[t] = map[string]interface{}{
								"File":     file,
								"Filename": fileHeaders[0].Filename,
							}
						}
					}
				}
			}

			for _, v := range data {
				if v == nil {
					render(w, r, "submit", map[string]interface{}{
						"Flash": "Make sure all fields are populated",
						"Items": config.SubmissionsItems,
					})
					return
				}
			}

			renderData := map[string]interface{}{
				"Items":   config.SubmissionsItems,
				"Success": true,
			}

			for t, d := range data {
				switch t {
				case "url":
					url := d.(string)
					err := google.SheetsSubmit(currentUser(r).TeamName(), url)
					if err != nil {
						panic(err)
					}
				case "file":
					file := d.(map[string]interface{})["File"].(io.Reader)
					filename := d.(map[string]interface{})["Filename"].(string)
					shareURL, err := google.DriveSubmit(currentUser(r).Info(), file, filename)
					if err != nil {
						panic(err)
					}
					renderData["ShareURL"] = shareURL
				}
			}

			render(w, r, "submit", renderData)
			return
		}

		render(w, r, "submit", map[string]interface{}{"Items": config.SubmissionsItems})
	}
}

func evaluation() (string, http.HandlerFunc) {
	return "/evaluation", func(w http.ResponseWriter, r *http.Request) {
		if !featureEnabled("evaluations") {
			http.NotFound(w, r)
			return
		}

		if !ensureLoggedIn(w, r) {
			return
		}

		if r.Method == http.MethodPost {
			r.ParseForm()

			slotID := strings.TrimSpace(r.FormValue("slot[id]"))

			if err := google.CalendarReserveTeamSlot(currentUser(r).TeamName(), slotID); err != nil {
				render(w, r, "evaluation", map[string]string{
					"Flash": err.Error(),
				})
			} else {
				http.Redirect(w, r, "/evaluation", http.StatusFound)
			}
			return
		}

		var teamSlot *Slot
		slot, err := google.CalendarTeamSlot(currentUser(r).TeamName())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if slot != nil {
			teamSlot = newSlotFromEvent(slot)
		}

		schedule := [][]*Slot{}

		currentDate := ""
		currentDay := -1
		slots, _ := google.CalendarFreeSlots()
		for _, slot := range slots {
			newSlot := newSlotFromEvent(slot)
			if currentDate != newSlot.Date {
				currentDate = newSlot.Date
				schedule = append(schedule, []*Slot{})
				currentDay++
			}

			schedule[currentDay] = append(schedule[currentDay], newSlot)
		}

		render(w, r, "evaluation", map[string]interface{}{
			"Schedule": schedule,
			"Reserved": teamSlot != nil,
			"Slot":     teamSlot,
		})
	}
}

func settings() (string, http.HandlerFunc) {
	return "/settings", func(w http.ResponseWriter, r *http.Request) {
		if !featureEnabled("settings") {
			http.NotFound(w, r)
			return
		}

		if !ensureLoggedIn(w, r) {
			return
		}

		render(w, r, "settings", map[string]interface{}{
			"SlackID": "",
		})
	}
}

func settingsSlack() (string, http.HandlerFunc) {
	return "/settings/slack", func(w http.ResponseWriter, r *http.Request) {
		if !featureEnabled("settings") {
			http.NotFound(w, r)
			return
		}

		if !ensureLoggedIn(w, r) {
			return
		}

		user := currentUser(r)

		success := fmt.Sprintf("Check your GUC email <code>%s</code> for an invitation.", user.Email())
		flash := ""

		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/settings", http.StatusFound)
			return
		}

		if err := slack.UsersAdminInvite(user.Email(), user.FirstName(), user.LastName()); err != nil {
			success, flash = "", fmt.Sprintf("Could not send invitation: %v", err)
		}

		render(w, r, "settings", map[string]string{
			"Success": success,
			"Flash":   flash,
		})
	}
}

func adminSessions() (string, http.HandlerFunc) {
	return "/admin/sessions", func(w http.ResponseWriter, r *http.Request) {
		if !ensureLoggedInAdmin(w, r) {
			return
		}

		render(w, r, "admin/sessions", map[string]interface{}{
			"Sessions": sessions,
		})
	}
}
