package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/ramin0/submit/config"
)

const (
	slackBaseURL = "https://slack.com/api"

	rChatPostEphemeral = "chat.postEphemeral"
	rChatPostMessage   = "chat.postMessage"
	rUsersAdminInvite  = "users.admin.invite"
	rUsersInfo         = "users.info"
	rUsersList         = "users.list"
	rRemindersAdd      = "reminders.add"
)

// ChatPostEphemeral func
func ChatPostEphemeral(user, message string) error {
	data := url.Values{}
	data.Add("channel", user)
	data.Add("text", message)
	data.Add("as_user", "true")

	if _, err := post(rChatPostEphemeral, data, config.SlackBotToken); err != nil {
		return err
	}

	return nil
}

// ChatPostMessage func
func ChatPostMessage(user, message string) error {
	data := url.Values{}
	data.Add("channel", user)
	data.Add("text", message)
	data.Add("as_user", "true")

	if _, err := post(rChatPostMessage, data, config.SlackBotToken); err != nil {
		return err
	}

	return nil
}

// UsersAdminInvite func
func UsersAdminInvite(email, firstName, lastName string) error {
	data := url.Values{}
	data.Add("email", email)
	data.Add("first_name", trim(firstName, 35))
	data.Add("last_name", trim(lastName, 35))
	data.Add("resend", "true")

	if _, err := post(rUsersAdminInvite, data, config.SlackTestToken); err != nil {
		return err
	}

	return nil
}

// UsersInfo func
func UsersInfo(user string) ([]string, error) {
	data := url.Values{}
	data.Add("user", user)

	json, err := post(rUsersInfo, data, config.SlackBotToken)
	if err != nil {
		return nil, err
	}

	u := json.S("user")

	username := u.S("name").Data()
	email := u.S("profile", "email").Data()

	return []string{
		fmt.Sprintf("%v", username),
		fmt.Sprintf("%v", email),
		fmt.Sprintf("%v", u.S("profile", "display_name").Data()),
		fmt.Sprintf("%v", u.S("real_name").Data()),
		fmt.Sprintf("%v", u.S("id").Data()),
	}, nil
}

// UsersList func
func UsersList(fn func([]string)) error {
	json, err := post(rUsersList, nil, config.SlackBotToken)
	if err != nil {
		return err
	}

	users, _ := json.S("members").Children()
	for _, u := range users {
		username := u.S("name").Data()
		email := u.S("profile", "email").Data()

		if email == nil {
			continue
		}

		fn([]string{
			fmt.Sprintf("%v", username),
			fmt.Sprintf("%v", email),
			fmt.Sprintf("%v", u.S("profile", "display_name").Data()),
			fmt.Sprintf("%v", u.S("real_name").Data()),
			fmt.Sprintf("%v", u.S("id").Data()),
		})
	}

	return nil
}

// RemindersAdd func
func RemindersAdd(user, text string, trigger time.Duration) error {
	data := url.Values{}
	data.Add("user", user)
	data.Add("text", text)
	data.Add("time", fmt.Sprintf("%d", time.Now().Add(trigger).Unix()))

	if _, err := post(rRemindersAdd, data, config.SlackUserToken); err != nil {
		return err
	}

	return nil
}

// WebhookResponse func
func WebhookResponse(url string, message interface{}) error {
	if _, err := post(url, nil, config.SlackBotToken, message); err != nil {
		return err
	}

	return nil
}

func post(method string, data url.Values, token string, body ...interface{}) (*gabs.Container, error) {
	if data == nil {
		data = url.Values{}
	}

	var requestBody io.Reader
	if len(data) > 0 {
		requestBody = strings.NewReader(data.Encode())
	} else if len(body) > 0 {
		b, err := json.Marshal(body[0])
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(b)
	}

	var url string
	if strings.HasPrefix(method, "http") {
		url = method
	} else {
		url = fmt.Sprintf("%s/%s?token=%s", slackBaseURL, method, token)
	}

	request, err := http.NewRequest(http.MethodPost, url, requestBody)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	} else if len(body) > 0 {
		request.Header.Add("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	jsonResponse, err := gabs.ParseJSONBuffer(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if success, ok := jsonResponse.S("ok").Data().(bool); ok && !success {
		if message, ok := jsonResponse.S("error").Data().(string); ok {
			return nil, fmt.Errorf(message)
		}

		return nil, fmt.Errorf("something went wrong")
	}

	return jsonResponse, nil
}

func trim(n string, l int) string {
	for len(n) > l {
		ns := strings.Fields(n)
		n = strings.Join(ns[:len(ns)-1], " ")
	}
	return n
}
