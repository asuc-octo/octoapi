package resourceslocation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/schema"
	"github.com/martinlindhe/unit"
	"github.com/umahmood/haversine"
	"google.golang.org/api/iterator"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()
var unitMap = map[string]unit.Length{
	"ft": unit.Foot,
	"yd": unit.Yard,
	"mi": unit.Mile,
	"m":  unit.Meter,
	"km": unit.Kilometer,
}

func ResourcesLocationEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	err := initFirestore(w)
	if err != nil {
		http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
		return
	}

	var input struct {
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
		Radius    float64 `json:"radius"`
		Unit      string  `json:"unit"`
	}
	err = decoder.Decode(&input, r.URL.Query())
	if err != nil || input.Longitude == 0 || input.Latitude == 0 || input.Radius == 0 || input.Unit == "" {
		http.Error(w, "Missing parameters or incorrect types", http.StatusBadRequest)
		return
	}
	if _, ok := unitMap[input.Unit]; !ok {
		http.Error(w, "Unsupported unit type", http.StatusBadRequest)
		return
	}

	resources, err := getResourceByRange(input.Longitude, input.Latitude, convertToKilometers(input.Radius, input.Unit))
	jsonString, err := json.Marshal(resources)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func getResourceByRange(longitude float64, latitude float64, radius float64) ([]map[string]interface{}, error) {

	defer client.Close()
	var resources []map[string]interface{}

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		docData := doc.Data()
		docLat, ok := docData["latitude"].(float64)
		docLon, ok := docData["longitude"].(float64)
		if !ok {
			log.Println("Error: Can't entry doesn't have location data available or is not of type float")
			continue
		}

		_, km := haversine.Distance(haversine.Coord{Lat: latitude, Lon: longitude},
			haversine.Coord{Lat: docLat, Lon: docLon})

		delete(docData, "latitude")
		if km <= radius {
			resources = append(resources, docData)
		}
	}
	return resources, nil
}

func convertToKilometers(value float64, units string) float64 {
	switch units {
	case "ft":
		return (unit.Length(value) * unit.Foot).Kilometers()
	case "yd":
		return (unit.Length(value) * unit.Yard).Kilometers()
	case "mi":
		return (unit.Length(value) * unit.Mile).Kilometers()
	case "m":
		return (unit.Length(value) * unit.Meter).Kilometers()
	case "km":
		return (unit.Length(value) * unit.Kilometer).Kilometers()
	}
	return 0.0
}
