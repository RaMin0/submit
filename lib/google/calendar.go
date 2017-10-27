package google

import (
	"fmt"
	"time"

	"github.com/ramin0/submit/config"
	calendar "google.golang.org/api/calendar/v3"
)

var (
	_calendarService *calendar.Service
)

func calendarService() (*calendar.Service, error) {
	if _calendarService == nil {
		c, err := googleClient()
		if err != nil {
			return nil, err
		}

		_calendarService, err = calendar.New(c)
		if err != nil {
			return nil, err
		}
	}

	return _calendarService, nil
}

// CalendarFreeSlots func
func CalendarFreeSlots() ([]*calendar.Event, error) {
	service, err := calendarService()
	if err != nil {
		return nil, err
	}

	timeMin, _ := time.Parse(time.RFC3339, config.EvaluationsWeekStart)
	timeMax, _ := time.Parse(time.RFC3339, config.EvaluationsWeekEnd)
	timeNow := time.Now()
	if timeNow.After(timeMin) {
		timeMin = timeNow
	}

	slots, err := service.Events.
		List(config.EvaluationsCalendarID).
		SingleEvents(true).
		OrderBy("startTime").
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		Q("FREE").
		Do()
	if err != nil {
		return nil, err
	}

	return slots.Items, nil
}

// CalendarTeamSlot func
func CalendarTeamSlot(teamName string) (*calendar.Event, error) {
	service, err := calendarService()
	if err != nil {
		return nil, err
	}

	slots, err := service.Events.
		List(config.EvaluationsCalendarID).
		SingleEvents(true).
		MaxResults(1).
		TimeMin(config.EvaluationsWeekStart).
		TimeMax(config.EvaluationsWeekEnd).
		Q(teamName).
		Do()
	if err != nil {
		return nil, err
	}

	if len(slots.Items) == 0 {
		return nil, nil
	}

	return slots.Items[0], nil
}

// CalendarReserveTeamSlot func
func CalendarReserveTeamSlot(teamName, slotID string) error {
	service, err := calendarService()
	if err != nil {
		return err
	}

	newSlot, err := service.Events.Get(config.EvaluationsCalendarID, slotID).Do()
	if err != nil {
		return err
	}

	if newSlot.Summary != "FREE" {
		return fmt.Errorf("slot already reserved")
	}

	oldSlot, _ := CalendarTeamSlot(teamName)

	newSlot = &calendar.Event{
		Summary: teamName,
		ColorId: "5",
	}
	if _, err := service.Events.Patch(config.EvaluationsCalendarID, slotID, newSlot).Do(); err != nil {
		return err
	}

	if oldSlot != nil {
		oldSlotID := oldSlot.Id
		oldSlot = &calendar.Event{
			Summary: "FREE",
			ColorId: "0",
		}
		_, err = service.Events.Patch(config.EvaluationsCalendarID, oldSlotID, oldSlot).Do()
		return err
	}

	return nil
}
