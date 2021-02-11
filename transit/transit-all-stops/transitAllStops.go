package transitallstops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	"github.com/martinlindhe/unit"
)

var apiURL = "http://api.actransit.org/transit/stops/{latitude}/{latitude}/{distance}"
var berkeleyLat = "37.871853"
var berkeleyLon = "-122.258423"
var defaultDistance = 3.0
var decoder = schema.NewDecoder()

func TransitAllStopsEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	// Read in query parameters
	var input struct {
		Longitude string  `json:"longitude"`
		Latitude  string  `json:"latitude"`
		Radius    float64 `json:"radius"`
		Unit      string  `json:"unit"`
	}
	err := decoder.Decode(&input, r.URL.Query())

	// Set default query parameters
	if err != nil {
		http.Error(w, "Error parsing parameters: "+err.Error(), http.StatusBadRequest)
		return
	}
	if input.Longitude == "" || input.Latitude == "" {
		input.Longitude = berkeleyLon
		input.Latitude = berkeleyLat
	}

	if input.Radius == 0 || input.Unit == "" {
		input.Radius = defaultDistance
		input.Unit = "mi"
	}

	// Read Transit API Key from Secrets Manager
	key, err := getTransitSecret(w)
	if err != nil {
		http.Error(w, "Error retrieving api key: "+err.Error(), http.StatusInternalServerError)
		return
	}
	stops, err := getAllStops(w, input.Longitude, input.Latitude, convertToFeet(input.Radius, input.Unit), key)
	if err != nil {
		http.Error(w, "error retrieving stops"+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonString, err := json.Marshal(stops)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func getAllStops(w http.ResponseWriter, longitude string, latitude string, radius float64, key string) ([]map[string]interface{}, error) {
	requestURL := strings.ReplaceAll(apiURL, "{latitude}", latitude)
	requestURL = strings.ReplaceAll(requestURL, "{longitude}", longitude)
	distance := strconv.FormatInt(int64(radius), 10)
	requestURL = strings.ReplaceAll(requestURL, "{distance}", distance)
	requestURL = requestURL + "?token=" + key

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stops []map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&stops)
	if err != nil {
		return nil, err
	}
	return stops, nil
}

func convertToFeet(value float64, units string) float64 {
	switch units {
	case "ft":
		return (unit.Length(value) * unit.Foot).Feet()
	case "yd":
		return (unit.Length(value) * unit.Yard).Feet()
	case "mi":
		return (unit.Length(value) * unit.Mile).Feet()
	case "m":
		return (unit.Length(value) * unit.Meter).Feet()
	case "km":
		return (unit.Length(value) * unit.Kilometer).Feet()
	}
	return 0.0
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}
