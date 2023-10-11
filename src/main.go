package main

/* CheckIn Bot automates the gathering of employee review documentation
by querying various datasources for performance information and
composing an email to the manager.
Ryan McNair Oct 9 2023

Pre-Reqs:
Configure AWS SES
Configure AWS account credentials

Usage:
Issue an HTTP GET request to https://<YOUR_API_ID>.execute-api.<YOUR_REGION>.amazonaws.com/bot?initials=<MANAGER_INITIALS>

Learnings:
1. Create public variables by capitalizing the first letter of the variable name
2. struct data collection type is pretty cool
3. := is a declaration AND assignment operator
4. _ underscore definition "The blank identifier may be used whenever syntax requires a variable name but program logic does not, for instance to discard an unwanted loop index when we require only the element value." From The Go Programming Language (Addison-Wesley Professional Computing Series) by Brian W. Kernighan
5. Functions can have defined and typed return variables, enabling "naked returns"
6. byte slices, woah
7. Golang uses package names at the top of each code file, instead of explicit class file include statements
8. When using multiple code files, the run command changes to "go run ."
9. Go does not allow importing unused dependencies, to the point that the IDE with GO SDK will automatically remove unused components from the import statement upon file save
10. Return functions are confusing (Return a function from a function), tried it, then removed as too complex.
*/

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Public constants
var SESRegion = "us-west-2"
var CurrentDate = time.Now().Format("January 2, 2006")
var ToEmail = ""
var FromEmail = ""
var GoalsURL = ""
var FeedbackURL = ""
var ManagerInitials = ""

// Private constants
var jiraAPIURL = ""
var jiraEmail = ""
var apiToken = ""

// Data for all configuration params
type BotConfig struct {
	ToEmail         string `json:"ToEmail"`
	ManagerInitials string `json:"ManagerInitials"`
	FromEmail       string `json:"FromEmail"`
	JiraEmail       string `json:"jiraEmail"`
	ApiToken        string `json:"apiToken"`
	JiraAPIURL      string `json:"jiraAPIURL"`
	GoalsURL        string `json:"GoalsURL"`
	FeedbackURL     string `json:"FeedbackURL"`
}

// Data for one Jira Issue
type JiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

// Data for one Employee Goal
type Goal struct {
	ID          int    `json:"ID"`
	Description string `json:"Description"`
	GoalDate    string `json:"GoalDate"`
	Achieved    string `json:"Achieved"`
	PushToDate  string `json:"PushToDate"`
}

// Data for one Employee with a set of Goals, currently expects only one Employee.
type Employee struct {
	Name         string `json:"Name"`
	Title        string `json:"Title"`
	Introduction string `json:"Introduction"`
	HireDate     string `json:"HireDate"`
	Goals        []Goal `json:"Goals"`
}

// Data is structured as a set of employees, for easier future extensibility, however currently expects only one Employee and at index zero.
type Employees struct {
	EmployeeList []Employee `json:"Employees"`
}

// This handler function is how AWS Lambda works
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// lookup config params
	config := getConfig()

	// set constants
	ToEmail = config.ToEmail
	FromEmail = config.FromEmail
	GoalsURL = config.GoalsURL
	FeedbackURL = config.FeedbackURL
	ManagerInitials = config.ManagerInitials
	jiraAPIURL = config.JiraAPIURL
	jiraEmail = config.JiraEmail
	apiToken = config.ApiToken

	// Authentication and authorization control: evaluate the initials query param before doing more things
	if strings.ToLower(request.QueryStringParameters["initials"]) != ManagerInitials {
		log.Println("Authorization error.") //Log an error
		response := events.APIGatewayProxyResponse{
			StatusCode: 403,
			Body:       fmt.Sprintf("Authorization error."),
			Headers: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				"Access-Control-Allow-Methods":     "GET,OPTIONS",
				"Access-Control-Allow-Credentials": "true",
			},
		}
		return response, nil
	} // otherwise continue with the program below

	// Read goals info from JSON file
	fileData, err := readFileFromGitHub(GoalsURL)
	if err != nil {
		fmt.Println("Error reading file:", err)
	}
	var employees Employees //Expects only 1 employee in this initial version
	err = json.Unmarshal(fileData, &employees)
	if err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
	}

	//Read employee feedback from text file
	fileText, err := readFileFromGitHub(FeedbackURL)
	if err != nil {
		log.Fatal("Error reading the file:", err)
	}

	// Compose email in paragraph form
	emailBody := "" //initialize the email body

	//Loop through the json file for employee info and goals
	for _, emp := range employees.EmployeeList {
		emailBody += fmt.Sprintf("%s\n\n", CurrentDate) //Today's date because this is an on-demand document
		emailBody += fmt.Sprintf("Name: %s\nTitle: %s\n", emp.Name, emp.Title)
		emailBody += fmt.Sprintf("Employment Duration: %s years\n\n", employmentDuration(emp.HireDate))
		emailBody += fmt.Sprintf("%s\n\n", emp.Introduction) //Intro paragraph
		emailBody += fmt.Sprintf("Profesional Development Goals:\n\n")
		for _, goal := range emp.Goals {
			emailBody += fmt.Sprintf("%s\nGoal Date: %s\nAchieved: %s\nPush To Date: %s\n\n", goal.Description, goal.GoalDate, goal.Achieved, goal.PushToDate)
		}
		emailBody += "\n"
	}

	//Add employee Q&A to the email
	emailBody += fmt.Sprintf("Talent Check-In Feedback:\n\n%s\n\n", fileText)

	//Add recently completed Jiras
	emailBody += fmt.Sprintf("Recent Accomplishments:\n")
	emailBody += fmt.Sprintf(getJiras(10, `project = ARCHITECT AND assignee = 60a590c19da6d50070cadedb AND status = Done and resolved is not null order by resolved desc`))

	//Add work in process Jiras
	emailBody += fmt.Sprintf("My Work In-Process:\n")
	emailBody += fmt.Sprintf(getJiras(5, `project = ARCHITECT AND assignee = 60a590c19da6d50070cadedb AND status = "In Progress" order by rank`))

	//Add email signature
	emailBody += fmt.Sprintf("\nSincerely,\n\n%s", employees.EmployeeList[0].Name)

	// Send email using AWS SES
	response, err := sendEmail(emailBody)
	return response, err
}

func main() {
	lambda.Start(handler)
}
