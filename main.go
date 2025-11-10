package main 

import (
	"fmt"
	"os"
	"log"
	"net/http"
	"io/ioutil"
)

func main() {
 weather_api_key := os.Getenv("WEATHER_API_KEY")
 latitude := 34.0008
 longitude := 81.0351

 response, err := http.Get(fmt.Sprintf("https://api.openweathermap.org/data/3.0/onecall?lat=%f&lon=%f&appid=%s", latitude, longitude, weather_api_key))

 if err != nil {
	 log.Fatal(err)
 }
 defer response.Body.Close()

 body, err := ioutil.ReadAll(response.Body)
 if err != nil {
	 log.Fatal(err)
 }
	
 fmt.Println(string(body))

}

