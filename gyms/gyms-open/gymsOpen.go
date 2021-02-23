package gymsopen

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/schema"
	"google.golang.org/api/iterator"
)

// type Timing struct {
//     Open_Time int64 `json:"open_time"`
//     Close_Time int64 `json:"close_time"`
// }

// type Gym struct {
//     Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Phone string `json:"phone"`
//     Description string `json:"description"`
//     Open_Close_Hours []Timing `json:"open_close_hours"`
//     Track_Hours []Timing `json:"track_hours"`
//     Pool_Hours []Timing `json:"pool_hours"`
// }

var decoder = schema.NewDecoder()
var client *firestore.Client
var ctx context.Context
var GymFields = [...]string{"name", "description", "latitude", "longitude", "address", "phone", "open_close_array", "track_hours", "pool_hours"}

func GymOpenEndpoint(w http.ResponseWriter, r *http.Request) {
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

	var timestamp int64
	var err error
	timestamp = time.Now().Unix()
	timeInput, ok := r.URL.Query()["time"]
	if ok {
		if len(timeInput[0]) >= 1 {
			timestamp, err = strconv.ParseInt(timeInput[0], 10, 64)
			if err != nil {
				http.Error(w, "Url Param 'time' is of incorrect type", http.StatusBadRequest)
				return
			}
		} else if len(timeInput[0]) < 1 {
			http.Error(w, "Url Param 'time' is of incorrect type", http.StatusBadRequest)
			return
		}
	}
	err = initFirestore(w)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", err)
		return
	}
	// Distance range
	var output []byte
	var gyms []map[string]interface{}
	gyms, err = getGymsOpen(w, timestamp)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Get Gyms Open failed: %v", err)
		return
	}
	output, err = json.Marshal(gyms)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Couldn't convert gym to JSON: %v", err)
		return
	}
	fmt.Fprint(w, string(output))
}

// radius in meters
func getGymsOpen(w http.ResponseWriter, timestamp int64) ([]map[string]interface{}, error) {
	defer client.Close()
	gyms := make([]map[string]interface{}, 0)
	iter := client.Collection("Gyms").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docData := doc.Data()
		gym := make(map[string]interface{})
		for _, element := range GymFields {
			gym[element] = docData[element]
		}
		hours, ok := gym["open_close_array"].([]map[string]interface{})
		if !ok {
			log.Printf("Couldn't parse open_close_array: %v", gym)
		} else {
			for _, timing := range hours {
				var open_time int64
				open_time, ok = timing["open_time"].(int64)
				if !ok {
					log.Printf("Couldn't parse open_time: %v", gym)
				} else {
					var close_time int64
					close_time, ok = timing["close_time"].(int64)
					if !ok {
						log.Printf("Couldn't parse close_time: %v", gym)
					} else {
						if open_time <= timestamp && timestamp <= close_time {
							gyms = append(gyms, gym)
							break
						}
					}
				}
			}
		}
	}
	return gyms, nil
}
