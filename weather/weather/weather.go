package weather

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func WeatherEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	apiKey, err := getWeatherSecret(w)
	if err != nil {
		http.Error(w, "Couldn't connect to Weather API", http.StatusInternalServerError)
		log.Printf("Weather Secret loading failed: %v", err)
		return
	}
	resp, err := http.Get("https://api.openweathermap.org/data/2.5/onecall?lat=37.8712&lon=-122.2601&appid=" + apiKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Weather API error: %v", err)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Couldn't connect to Weather API", http.StatusInternalServerError)
		log.Printf("Couldn't convert Weather API output to our output: %v", err)
		return
	}
	fmt.Fprint(w, string(body))
}
