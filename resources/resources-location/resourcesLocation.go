package resourceslocation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/schema"
	"github.com/martinlindhe/unit"
	"github.com/umahmood/haversine"
	"google.golang.org/api/iterator"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()
var floatType = reflect.TypeOf(float64(0))

var unitMap = map[string]unit.Length{
	"ft": unit.Foot,
	"yd": unit.Yard,
	"mi": unit.Mile,
	"m":  unit.Meter,
	"km": unit.Kilometer,
}

func ResourcesLocationEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,PATCH,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,PATCH,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid Access Token: Make sure you are passing in an access token in the header of your request using bearer token authentication. To get your token please visit the Getting Started section on our API documentation page. Access tokens expire within 2 days, so make sure you retrieve your new valid access token using the refresh_token endpoint.", http.StatusBadRequest)
		return
	}

	var radius float64
	var longitude float64
	var latitude float64
	var units string
	var err error
	radiusInput, ok := r.URL.Query()["radius"]
	if ok {
		if len(radiusInput[0]) >= 1 {
			radius, err = strconv.ParseFloat(radiusInput[0], 64)
			if err != nil {
				http.Error(w, "Url Param 'radius' is of incorrect type", http.StatusBadRequest)
				return
			}
		} else if len(radiusInput[0]) < 1 {
			http.Error(w, "Url Param 'radius' is of incorrect type", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Url Param 'radius' is missing", http.StatusBadRequest)
		return
	}
	longitudeInput, ok := r.URL.Query()["longitude"]
	if ok {
		if len(longitudeInput[0]) >= 1 {
			longitude, err = strconv.ParseFloat(longitudeInput[0], 64)
			if err != nil {
				http.Error(w, "Url Param 'longitude' is of incorrect type", http.StatusBadRequest)
				return
			}
		} else if len(longitudeInput[0]) < 1 {
			http.Error(w, "Url Param 'longitude' is of incorrect type", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Url Param 'longitude' is missing", http.StatusBadRequest)
		return
	}

	latitudeInput, ok := r.URL.Query()["latitude"]
	if ok {
		if len(latitudeInput[0]) >= 1 {
			latitude, err = strconv.ParseFloat(latitudeInput[0], 64)
			if err != nil {
				http.Error(w, "Url Param 'latitude' is of incorrect type", http.StatusBadRequest)
				return
			}
		} else if len(latitudeInput[0]) < 1 {
			http.Error(w, "Url Param 'latitude' is of incorrect type", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Url Param 'latitude' is missing", http.StatusBadRequest)
		return
	}

	unitsInput, ok := r.URL.Query()["unit"]
	if !ok || len(unitsInput) < 1 {
		http.Error(w, "Url Param 'unit' is missing", http.StatusBadRequest)
		return
	}
	units = unitsInput[0]

	var kilometers float64
	kilometers, err = convertToKilometers(radius, units)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = initFirestore(w)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	resources, err := getResourceByRange(longitude, latitude, kilometers)
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

func convertToKilometers(value float64, units string) (float64, error) {
	unitConvert, valid := unitMap[units]
	if valid {
		return (unit.Length(value) * unitConvert).Kilometers(), nil
	} else {
		return 0, errors.New("URL Param 'unit' is incorrect")
	}
}

func getFloat(unk interface{}) (float64, error) {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return 0, fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
}
