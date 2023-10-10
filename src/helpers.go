package main

// This file contains all the helper functions for Check-In Bot.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ses"
)

// Retrieves all config params from AWS Secrets Manager
func getConfig() BotConfig {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(SESRegion),
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("CheckInBotConfig"),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		log.Fatalf("Failed to retrieve secret: %v", err)
	}

	var config BotConfig
	err = json.Unmarshal([]byte(*result.SecretString), &config)
	if err != nil {
		log.Fatalf("Failed to unmarshal secrets: %v", err)
	}
	return config
}

// Translates a hire date string into a decimal number of years
func employmentDuration(hireDateString string) string {
	// 1. Parse the HireDate string into a time.Time value.
	hireDate, err := time.Parse("2006-01-02", hireDateString)
	if err != nil {
		log.Fatalf("Failed to parse date: %v", err)
	}

	// 2. Subtract the hire date from the current date to get the duration.
	now := time.Now()
	duration := now.Sub(hireDate)

	// 3. Calculate the number of years and remaining months.
	years := int(duration.Hours() / 24 / 365)
	remainingMonths := int(duration.Hours()/24/30) % 12

	// 4. Convert remaining months into a fraction of a year.
	fractionYear := float64(remainingMonths) / 12.0

	// 5. Sum the whole years with the fraction.
	totalYears := float64(years) + fractionYear

	return fmt.Sprintf("%.1f", totalYears)
}

// This method actually calls the Jira API
func queryJiraIssues(jql, apiURL, apiUser, apiToken string) []JiraIssue {
	queryURL := apiURL + "?jql=" + jql
	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Using Basic Authentication with the provided API token:
	req.SetBasicAuth(apiUser, apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var issuesResponse struct {
		Issues []JiraIssue `json:"issues"`
	}

	err = json.Unmarshal(body, &issuesResponse)
	if err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	return issuesResponse.Issues
}

// Gets a specified number of Jira Issues Key and Summary from Jira API
func getJiras(maxItems int, jql string) string {
	encodedJQL := url.QueryEscape(jql)
	jiraIssues := queryJiraIssues(encodedJQL, jiraAPIURL, jiraEmail, apiToken)

	var jiraParagraph = "\n"

	for i := 0; i < len(jiraIssues) && i < maxItems; i++ {
		jiraParagraph += fmt.Sprintf("%s: %s\n", jiraIssues[i].Key, jiraIssues[i].Fields.Summary)
	}

	jiraParagraph += "\n"

	return jiraParagraph
}

// Send an email via AWS SES with custom message body
func sendEmail(emailBody string) (events.APIGatewayProxyResponse, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(SESRegion)},
	)
	if err != nil {
		log.Println("Failed to create AWS session:", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to create AWS session: %s", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				"Access-Control-Allow-Methods":     "GET,OPTIONS",
				"Access-Control-Allow-Credentials": "true",
			},
		}
		return response, err
	}
	svc := ses.New(sess)
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(ToEmail),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Data: aws.String(emailBody),
				},
			},
			Subject: &ses.Content{
				Data: aws.String("Here is the Talent Check-In you requested"),
			},
		},
		Source: aws.String(FromEmail),
	}
	_, err = svc.SendEmail(input)
	if err != nil {
		log.Println("Failed to send email:", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to send email: %s", err),
			Headers: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				"Access-Control-Allow-Methods":     "GET,OPTIONS",
				"Access-Control-Allow-Credentials": "true",
			},
		}
		return response, err
	}
	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Email sent successfully",
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
			"Access-Control-Allow-Methods":     "GET,OPTIONS",
			"Access-Control-Allow-Credentials": "true",
		},
	}
	return response, err
}

// Read a file from GitHub
func readFileFromGitHub(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch content, status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}
