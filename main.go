package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/valyala/fastjson"

	"context"
	"encoding/json"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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
		LogErr(err)
		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			LogErr(err)
		}
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
		err := f.Close()
		if err != nil {
			LogErr(err)
		}
	}()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		LogErr(err)
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
	// building prompt
	ai_prompt.WriteString("Here is the daily weather forecast for Columbia, SC:\n")
	ai_prompt.WriteString("Daily summary: " + m["daily_summary"] + "\n")
	ai_prompt.WriteString("Daily High: " + m["daily_max"] + "\u00B0F\n")
	ai_prompt.WriteString("Daily Low: " + m["daily_low"] + "\u00B0F\n")
	ai_prompt.WriteString("Daily Humidity: " + m["daily_humidity"] + "%\n")
	ai_prompt.WriteString("Current humidity " + m["current_humidity"] + "%\n")
	ai_prompt.WriteString("Current feels like: " + m["current_feels_like"] + "\u00B0F\n")

	fmt.Println(ai_prompt.String())

	// testing Google provided code

	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
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

	//TODO: get credentials setup for google calendar api
	// get all events from today and list times

	// call calendar func

	// have ai figure out if i have anything important like exams or things other than school (based on response I feed it)

	// FUTURE: setup NLP to process calendar changes on my phone
	// check gemini chats
	// will have to do this using python for ease of use
}
