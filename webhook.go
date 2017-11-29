package submit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-errors/errors"
	"github.com/ramin0/submit/config"
	"github.com/ramin0/submit/lib/google"
	"github.com/ramin0/submit/lib/slack"
	"github.com/ramin0/submit/lib/util"
)

var (
	slackIDRegexp = regexp.MustCompile("^<@(.+)\\|.+>$")
)

type payload struct {
	Token     string
	Challenge string
	Event     event
}

type event struct {
	Type string
	User struct {
		ID       string
		UserName string `json:"name"`
		Name     string `json:"real_name"`
	}
	Command struct {
		Cmd         string
		Text        string
		ResponseURL string
		Channel     struct {
			ID   string
			Name string
		}
		User struct {
			ID string
		}
	}
}

func webhook() (string, http.HandlerFunc) {
	return "/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			renderJSON(w, map[string]interface{}{"ok": false}, http.StatusMethodNotAllowed)
			return
		}

		body := decodePayload(r)

		if body.Challenge != "" {
			fmt.Fprint(w, body.Challenge)
			return
		}

		if body.Token != config.SlackWebhookToken {
			renderJSON(w, map[string]interface{}{"ok": false}, http.StatusUnauthorized)
			return
		}

		if err := handleWebhook(body.Event); err != nil {
			renderJSON(w, map[string]interface{}{"ok": false, "error": err.Error()})
			return
		}

		renderJSON(w, map[string]interface{}{
			"ok": true,
		})
	}
}

func decodePayload(r *http.Request) (body payload) {
	defer r.Body.Close()

	switch contentType := r.Header.Get("Content-Type"); contentType {
	case "application/x-www-form-urlencoded":
		r.ParseForm()
		body.Token = r.FormValue("token")
		body.Event.Type = "cmd"
		body.Event.Command.Cmd = r.FormValue("command")[1:]
		body.Event.Command.Text = r.FormValue("text")
		body.Event.Command.ResponseURL = r.FormValue("response_url")
		body.Event.Command.Channel.ID = r.FormValue("channel_id")
		body.Event.Command.Channel.Name = r.FormValue("channel_name")
		body.Event.Command.User.ID = r.FormValue("user_id")
	case "application/json":
		json.NewDecoder(r.Body).Decode(&body)
	}

	return body
}

func handleWebhook(e event) error {
	switch e.Type {
	case "cmd":
		switch e.Command.Cmd {
		case "id":
			return handleCommandID(e)
		case "team":
			return handleCommandTeam(e)
		case "proposal":
			return handleCommandProposal(e)
		}
	}

	return nil
}

func handleCommandID(e event) error {
	slackID := e.Command.Text
	if slackID == "" {
		return fmt.Errorf("Missing User ID")
	}
	if !slackIDRegexp.MatchString(slackID) {
		return fmt.Errorf("Invalid Argument: %v", slackID)
	}
	slackID = slackIDRegexp.FindStringSubmatch(slackID)[1]

	if !slackAdminsRegexp().MatchString(e.Command.User.ID) {
		if slackID != e.Command.User.ID {
			return fmt.Errorf("Unauthorized")
		}
	}

	go func(slackID string) {
		defer func() {
			if err := recover(); err != nil {
				panicHandler(nil, nil, errors.Wrap(err, 1))
			}
		}()

		info, err := slack.UsersInfo(slackID)
		if err != nil {
			panic(err)
		}

		student, err := google.SheetsUserInfoBy("Email", info[1])
		if err != nil {
			panic(err)
		}

		slack.WebhookResponse(e.Command.ResponseURL, map[string]interface{}{
			"attachments": []interface{}{
				map[string]interface{}{
					"title": student["FullName"],
					"text":  student["Email"],
					"fields": []map[string]interface{}{
						map[string]interface{}{
							"title": "ID",
							"value": student["ID"],
							"short": true,
						},
						map[string]interface{}{
							"title": "Tutorial Group",
							"value": student["Group"],
							"short": true,
						},
						map[string]interface{}{
							"title": "Team",
							"value": student["Team"],
							"short": true,
						},
						map[string]interface{}{
							"title": "Team Tutorial Group",
							"value": student["TeamGroup"],
							"short": true,
						},
					},
				},
			},
		})
	}(slackID)

	return nil
}

func handleCommandTeam(e event) error {
	teamID := e.Command.Text
	if teamID == "" {
		return fmt.Errorf("Missing Team ID")
	}

	if !slackAdminsRegexp().MatchString(e.Command.User.ID) {
		info, err := slack.UsersInfo(e.Command.User.ID)
		if err != nil {
			return err
		}
		user, err := google.SheetsUserInfoBy("Email", info[1])
		if err != nil {
			return err
		}

		if util.TrimTeamName(user["Team"]) != teamID {
			return fmt.Errorf("Unauthorized")
		}
	}

	go func(teamID string) {
		defer func() {
			if err := recover(); err != nil {
				panicHandler(nil, nil, errors.Wrap(err, 1))
			}
		}()

		teamName := util.FormatTeamName(teamID)

		members, err := google.SheetsTeamMembers(teamName)
		if err != nil {
			panic(err)
		}

		fields := []map[string]interface{}{
			map[string]interface{}{
				"title": "Team",
				"value": teamName,
				"short": true,
			},
			map[string]interface{}{
				"title": "Team Tutorial Group",
				"value": members[0]["TeamGroup"],
				"short": true,
			},
		}

		for _, m := range members {
			fields = append(fields, map[string]interface{}{
				"title": fmt.Sprintf("[%s] %s (%s)", m["ID"], m["FullName"], m["Group"]),
				"value": fmt.Sprintf("%s@student.guc.edu.eg", m["UserName"]),
				"short": false,
			})
		}

		slack.WebhookResponse(e.Command.ResponseURL, map[string]interface{}{
			"attachments": []interface{}{
				map[string]interface{}{
					"fields": fields,
				},
			},
		})
	}(teamID)

	return nil
}

func handleCommandProposal(e event) error {
	teamID := e.Command.Text
	if teamID == "" {
		return fmt.Errorf("Missing Team ID")
	}

	if !slackAdminsRegexp().MatchString(e.Command.User.ID) {
		info, err := slack.UsersInfo(e.Command.User.ID)
		if err != nil {
			return err
		}
		user, err := google.SheetsUserInfoBy("Email", info[1])
		if err != nil {
			return err
		}

		if util.TrimTeamName(user["Team"]) != teamID {
			return fmt.Errorf("Unauthorized")
		}
	}

	go func(teamID string) {
		defer func() {
			if err := recover(); err != nil {
				panicHandler(nil, nil, errors.Wrap(err, 1))
			}
		}()

		teamName := util.FormatTeamName(teamID)

		proposal, err := google.SheetsTeamProposal(teamName)
		if err != nil {
			panic(err)
		}

		fields := []map[string]interface{}{
			map[string]interface{}{
				"title": "Team",
				"value": teamName,
				"short": true,
			},
		}

		for _, qa := range proposal["QAs"].([][]string) {
			fields = append(fields, map[string]interface{}{
				"title": qa[0],
				"value": qa[1],
				"short": false,
			})
		}

		slack.WebhookResponse(e.Command.ResponseURL, map[string]interface{}{
			"attachments": []interface{}{
				map[string]interface{}{
					"fields": fields,
				},
			},
		})
	}(teamID)

	return nil
}

func slackAdminsRegexp() *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("(?:%s)", strings.Join(config.SlackAdmins, "|")))
}
