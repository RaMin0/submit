package submit

import (
	"fmt"
	"strings"
	"time"

	"github.com/ramin0/submit/lib/google"
)

// Session struct
type Session struct {
	Timestamp time.Time
	History   []string
	User      *User
}

// User struct
type User struct {
	ID          string
	UserName    string
	FullName    string
	group       string
	teamName    string
	teamGroup   string
	teamMembers []*User
	proposal    map[string]interface{}
}

// FirstName func
func (user *User) FirstName() string {
	return strings.Fields(user.FullName)[0]
}

// LastName func
func (user *User) LastName() string {
	return strings.Join(strings.Fields(user.FullName)[1:], " ")
}

// Email func
func (user *User) Email() string {
	if user.Admin() {
		return fmt.Sprintf("%s@guc.edu.eg", user.UserName)
	}

	return fmt.Sprintf("%s@student.guc.edu.eg", user.UserName)
}

// Group func
func (user *User) Group() string {
	if user.group == "" {
		user.fetchInfo()
	}

	return user.group
}

// TeamName func
func (user *User) TeamName() string {
	if user.teamName == "" {
		user.fetchInfo()
	}

	return user.teamName
}

// TeamGroup func
func (user *User) TeamGroup() string {
	if user.teamGroup == "" {
		user.fetchInfo()
	}

	return user.teamGroup
}

// TeamMembers func
func (user *User) TeamMembers() []*User {
	if user.teamMembers == nil {
		user.teamMembers = []*User{}

		if !user.Admin() {
			teamMembers, _ := google.SheetsTeamMembers(user.TeamName())
			for _, teamMember := range teamMembers {
				user.teamMembers = append(user.teamMembers, &User{
					ID:       teamMember["ID"],
					FullName: teamMember["FullName"],
					UserName: teamMember["UserName"],
				})
			}
		}
	}

	return user.teamMembers
}

// Proposal func
func (user *User) Proposal() map[string]interface{} {
	if user.proposal == nil {
		user.proposal, _ = google.SheetsTeamProposal(user.TeamName())
	}

	return user.proposal
}

// Info func
func (user *User) Info() map[string]string {
	return map[string]string{
		"ID":        user.ID,
		"UserName":  user.UserName,
		"FullName":  user.FullName,
		"Email":     user.Email(),
		"Group":     user.Group(),
		"Team":      user.TeamName(),
		"TeamGroup": user.TeamGroup(),
	}
}

// Admin func
func (user *User) Admin() bool {
	return user.group == "admins"
}

func (user *User) fetchInfo() error {
	info, err := google.SheetsUserInfoBy("ID", user.ID)
	if err != nil {
		return err
	}

	user.group = info["Group"]
	user.teamName = info["Team"]
	user.teamGroup = info["TeamGroup"]

	return nil
}

// Slot struct
type Slot struct {
	ID   string
	Date string
	Time string
}
