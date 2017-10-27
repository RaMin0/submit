package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ramin0/submit/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	drive "google.golang.org/api/drive/v3"
	sheets "google.golang.org/api/sheets/v4"
)

var (
	scopes = []string{
		calendar.CalendarScope,
		drive.DriveScope,
		sheets.SpreadsheetsScope,
	}
)

func googleClient() (*http.Client, error) {
	ctx := context.Background()

	b := []byte(config.GoogleAPIClientSecret)
	cfg, err := google.ConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, err
	}

	if config.GoogleAPIClientToken == "" {
		authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		fmt.Println(authURL)

		var code string
		fmt.Print("Code: ")
		if _, err := fmt.Scan(&code); err != nil {
			return nil, err
		}

		token, err := cfg.Exchange(oauth2.NoContext, code)
		if err != nil {
			return nil, err
		}

		if b, err := json.Marshal(token); err == nil {
			config.GoogleAPIClientToken = string(b)
			fmt.Println(config.GoogleAPIClientToken)
		}
	}

	token := oauth2.Token{}
	json.Unmarshal([]byte(config.GoogleAPIClientToken), &token)

	return cfg.Client(ctx, &token), nil
}
