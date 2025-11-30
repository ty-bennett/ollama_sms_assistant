package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/valyala/fastjson"

	"context"
	"encoding/json"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// email struct to easily read content later for ai prompt

type Email struct {
	Subject string
	Sender  string
	Snippet string
	Date    string
}

type CalendarEvent struct {
	Title    string
	Date     string
	Time     string
	Location string
}

type OllamaPrompt struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func LogErr(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		log.Fatal(err)
	}
}

func GetRecentEmails(srv *gmail.Service, limit int64) ([]Email, error) {
	// set list of emails (using struct)
	var emails []Email
	//set user as me (tybennett)
	user := "me"
	// get List of emails (returns Ids of emails so i can use them later)
	r, err := srv.Users.Messages.List(user).LabelIds("INBOX").MaxResults(limit).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %v", err)
	}
	// null check
	if len(r.Messages) == 0 {
		fmt.Println("No messages found.")
		return emails, nil
	}

	for _, m := range r.Messages {
		msg, err := srv.Users.Messages.Get(user, m.Id).Format("full").Do()
		if err != nil {
			log.Printf("Could not fetch message %v: %v", m.Id, err)
			continue
		}

		// Extract Headers (Subject, From)
		subject := ""
		sender := ""
		date := ""

		for _, h := range msg.Payload.Headers {
			switch h.Name {
			case "Subject":
				subject = h.Value
			case "From":
				sender = h.Value
			case "Date":
				date = h.Value
			}
		}
		emails = append(emails, Email{
			Subject: subject,
			Sender:  sender,
			Snippet: msg.Snippet, // Google automatically generates a summary snippet
			Date:    date,
		})
	}

	return emails, nil
}

func GetCalendarEvents(srv *calendar.Service) ([]CalendarEvent, error) {
	var calendar_events_list []CalendarEvent
	// 1. Calculate the Start and End of "Today"
	now := time.Now()
	// Start of day: 00:00:00
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format(time.RFC3339)
	// End of day: 23:59:59
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)

	calendar_list, err := srv.CalendarList.List().Do()

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve calendar list: %v", err)
	}

	for _, cal := range calendar_list.Items {

		events, err := srv.Events.List(cal.Id).ShowDeleted(false).SingleEvents(true).TimeMin(startOfDay).TimeMax(endOfDay).OrderBy("startTime").Do()
		if cal.Summary == "SCHOOL" {
			continue
		}
		if err != nil {
			log.Printf("Could not fetch events for calendar %s: %v", cal.Summary, err)
			continue
		}

		for _, item := range events.Items {
			date := item.Start.DateTime
			timeStr := ""

			if date == "" {
				date = item.Start.Date
				timeStr = "All Day"
			} else {
				// Parse the RFC3339 time to make it readable (e.g. "15:04")
				t, _ := time.Parse(time.RFC3339, date)
				timeStr = t.Format("3:04 PM") // Format as "3:00 PM"
				date = t.Format("2006-01-02") // Just the date part
			}
			title := fmt.Sprintf("[%s] %s", cal.Summary, item.Summary)
			calendar_events_list = append(calendar_events_list, CalendarEvent{
				Title:    title,
				Date:     date,
				Time:     timeStr,
				Location: item.Location,
			})
		}
	}
	return calendar_events_list, nil
}

func SendOllamaPrompt(promptData *OllamaPrompt) ([]byte, error) {
	prompt, err := json.Marshal(promptData)
	if err != nil {
		LogErr(err)
	}
	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewReader(prompt))
	if err != nil {
		log.Fatalf("error retrieving ollama response: %v", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("error retrieving ollama response: %v", err)
		}
	}()

	ai_response, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reading ollama response: %v", err)
	}
	return ai_response, nil
}

func main() {

	// load env variable for API key
	err := godotenv.Load()
	if err != nil {
		// if error occurs when loading log
		log.Fatal("Env file could not be loaded")
	}

	// set api key to the env varibale
	weather_api_key := os.Getenv("WEATHER_API_KEY")

	// Columbia, SC lat and long
	const latitude = 34.0007
	const longitude = -81.0348

	// Get request to API with correct Request Params sending those into response and err vars
	response, err := http.Get(fmt.Sprintf("https://api.openweathermap.org/data/3.0/onecall?lat=%f&lon=%f&appid=%s&exclude=minutely,hourly,alerts&units=imperial&lang=en", latitude, longitude, weather_api_key))

	// if that err is not nil then log error
	LogErr(err)
	//set body to read all response.Body
	body, err := io.ReadAll(response.Body)

	LogErr(err)

	var p fastjson.Parser

	raw_json, err := p.Parse(string(body))
	LogErr(err)

	fmt.Println("Fetching Weather...")

	daily_max := raw_json.Get("daily", "0", "temp", "max").GetFloat64()
	daily_low := raw_json.Get("daily", "0", "temp", "min").GetFloat64()
	daily_humidity := raw_json.Get("daily", "0", "humidity").GetFloat64()
	daily_summary := raw_json.Get("daily", "0", "summary").GetStringBytes()
	current_humidity := raw_json.Get("current", "humidity").GetFloat64()
	current_feels_like := raw_json.Get("current", "feels_like").GetFloat64()

	m := make(map[string]string)

	m["daily_max"] = strconv.FormatFloat(daily_max, 'f', -1, 64)
	m["daily_low"] = strconv.FormatFloat(daily_low, 'f', -1, 64)
	m["daily_humidity"] = strconv.FormatFloat(daily_humidity, 'f', -1, 64)
	m["daily_summary"] = string(daily_summary)
	m["current_humidity"] = strconv.FormatFloat(current_humidity, 'f', -1, 64)
	m["current_feels_like"] = strconv.FormatFloat(current_feels_like, 'f', -1, 64)

	// string builder to build the prompt
	var ai_prompt strings.Builder
	err = godotenv.Load("prompt.txt")
	if err != nil {
		log.Fatal("prompt could not be loaded")
	}
	prompt_from_file := os.Getenv("prompt.txt")
	ai_prompt.WriteString(prompt_from_file)

	// building prompt
	ai_prompt.WriteString("Here is the daily weather forecast for Columbia, SC:\n")
	ai_prompt.WriteString("Daily summary: " + m["daily_summary"] + "\n")
	ai_prompt.WriteString("Daily High: " + m["daily_max"] + "\u00B0F\n")
	ai_prompt.WriteString("Daily Low: " + m["daily_low"] + "\u00B0F\n")
	ai_prompt.WriteString("Daily Humidity: " + m["daily_humidity"] + "%\n")
	ai_prompt.WriteString("Current humidity " + m["current_humidity"] + "%\n")
	ai_prompt.WriteString("Current feels like: " + m["current_feels_like"] + "\u00B0F\n")

	// testing Google provided code

	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrive Gmail client: %v", err)
	}
	calendar_srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	// function call to get messages
	// get top 10 latest emails
	fmt.Println("Fetching latest emails...")
	list_of_emails, err := GetRecentEmails(srv, 10)
	if err != nil {
		LogErr(err)
	}

	ai_prompt.WriteString("\n--- Recent Emails ---\n")
	for _, e := range list_of_emails {
		ai_prompt.WriteString(fmt.Sprintf("From: %s\nSubject: %s\nSnippet: %s\n\n", e.Sender, e.Subject, e.Snippet))
	}
	fmt.Println("Fetching calendar events for today...")
	// We call our updated function that searches ALL calendars for TODAY only
	list_of_events, err := GetCalendarEvents(calendar_srv)
	if err != nil {
		LogErr(err)
	}

	// Add the Calendar section to the AI Prompt
	ai_prompt.WriteString("\n--- Today's Schedule ---\n")

	if len(list_of_events) == 0 {
		ai_prompt.WriteString("No events scheduled for today.\n")
	} else {
		for _, e := range list_of_events {
			// Format: [Work] Team Meeting @ 2:00 PM
			// or: [Holidays] Christmas @ All Day
			ai_prompt.WriteString(fmt.Sprintf("%s @ %s", e.Title, e.Time))

			// Only add location if it actually exists
			if e.Location != "" {
				ai_prompt.WriteString(fmt.Sprintf(" (Loc: %s)", e.Location))
			}
			ai_prompt.WriteString("\n")
		}
	}

	// Final Print to see what we are sending to Ollama
	fmt.Println("\n=== FINAL PROMPT TO SEND TO AI ===")
	fmt.Println(ai_prompt.String())
	fmt.Println("==================================")

	//TODO: format and send prompt to AI
	// have ai figure out if i have anything important like exams or things other than school (based on response I feed it)
	var req OllamaPrompt
	ollama_model := "llama3.1"
	req.Model = ollama_model
	req.Prompt = ai_prompt.String()
	req.Stream = false

	res, err := SendOllamaPrompt(&req)
	if err != nil {
		log.Fatalf("Error retrieving response from Ollama: %v", err)
	}
	var final_answer OllamaResponse
	err = json.Unmarshal(res, &final_answer)
	if err != nil {
		log.Fatalf("Error converting json values from ai response: %v", err)
	}

	fmt.Println(final_answer.Response)

	// FUTURE: setup NLP to process calendar changes on my phone
	// check gemini chats
	// will have to do this using python for ease of use
}
