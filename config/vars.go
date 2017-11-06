package config

// var
var (
	// Submit
	SubmitName         = "ACML"
	AdminPassword      = ""
	SubmissionDeadline = "1989-03-21T00:00:00+02:00"
	TeamNameFormat     = "Team %2v"
	FeaturesEnabled    = map[string]bool{}

	// Google
	GoogleAPIClientSecret      = ""
	GoogleAPIClientToken       = ""
	StudentsSheetID            = ""
	SubmissionsItems           = []map[string]string{}
	SubmissionsFolderID        = ""
	SubmissionsMetaDescription = "Uploaded By:\n- {{.FullName}}\n- {{.Email}}\n- {{.ID}}"
	EvaluationsCellRange       = "'Evaluations'!B%d"
	EvaluationsCalendarID      = "primary"
	EvaluationsWeekStart       = "1989-03-21T00:00:00+02:00"
	EvaluationsWeekEnd         = "1989-03-21T00:00:00+02:00"

	// Slack
	SlackTestToken    = ""
	SlackUserToken    = ""
	SlackBotToken     = ""
	SlackWebhookToken = ""
	SlackAdmins       = []string{}
)
