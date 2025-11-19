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
)

func LogErr(e error) {
	if e != nil {
		log.Fatal(e)
	}
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

	daily_feels_like := raw_json.Get("daily", "0", "feels_like", "day").GetFloat64()
	daily_humidity := raw_json.Get("daily", "0", "humidity").GetFloat64()
	daily_summary := raw_json.Get("daily", "0", "summary").GetStringBytes()
	// TODO:
	//	weather_summary := raw_json.Get("daily", "0", "summary").GetStringBytes()
	//  humidity := raw_json.Get("current")

	m := make(map[string]string)

	m["daily_feels_like"] = strconv.FormatFloat(daily_feels_like, 'f', -1, 64)
	m["daily_humidity"] = strconv.FormatFloat(daily_humidity, 'f', -1, 64)
	m["daily_summary"] = string(daily_summary)
	m["daily_feels_like"] = strconv.FormatFloat(daily_feels_like, 'f', -1, 64)

	var ai_response strings.Builder
	ai_response.WriteString("Here is the daily weather forecase for Columbia, SC:\n")
	ai_response.WriteString("Daily feels like temp: ")
	ai_response.WriteString(m["daily_feels_like"] + "\n")
	ai_response.WriteString("Daily high temp: ")
	ai_response.WriteString(m["daily_humidity"] + "\n")

}
