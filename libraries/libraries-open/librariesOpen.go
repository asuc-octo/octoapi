package librariesopen

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// type Timing struct {
//     Open_Time int64 `json:"open_time"`
//     Close_Time int64 `json:"close_time"`
// }

// type Library struct {
// 	Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Description string `json:"description"`
//     Address string `json:"address"`
//     Open_Close_Hours []Timing `json:"open_close_hours"`
// }

var LibraryFields = [6]string{"name", "description", "latitude", "longitude", "address", "open_close_array"}
var client *firestore.Client
var ctx context.Context

func LibraryOpenEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	fstoreErr := initFirestore(w)
	var timestamp int64
	currtime := r.URL.Query().Get("time")
	if currtime == "" {
		timestamp = time.Now().Unix()
	} else {
		var timeErr error
		timestamp, timeErr = strconv.ParseInt(currtime, 10, 64)
		if timeErr != nil {
			http.Error(w, timeErr.Error(), http.StatusBadRequest)
		}
	}
	if fstoreErr != nil {
		http.Error(w, "Couldn’t connect to database", http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", fstoreErr)
		return
	}
	libraries, libraryErr := openLibraries(ctx, w, client, timestamp)
	if libraryErr != nil {
		http.Error(w, "Couldn’t connect to database", http.StatusInternalServerError)
		log.Printf("libraries search GET failed: %v", libraryErr)
		return
	}
	output, jsonErr := json.Marshal(&libraries)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
		log.Printf("libraries JSON conversion failed: %v", jsonErr)
		return
	}
	fmt.Fprint(w, string(output))
}

func openLibraries(ctx context.Context, w http.ResponseWriter, client *firestore.Client, timestamp int64) ([]map[string]interface{}, error) {
	defer client.Close()
	libraries := make([]map[string]interface{}, 0)
	iter := client.Collection("Libraries").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docData := doc.Data()
		hours, hoursOk := docData["open_close_array"].([]interface{})
		if !hoursOk {
			continue
		} else {
			for _, timing := range hours {
				hour, hourOk := timing.(map[string]interface{})
				if !hourOk {
					continue
				}
				notes := hour["notes"].(string)
				openTime := int64(hour["open_time"].(float64))
				endTime := int64(hour["close_time"].(float64))
				if notes != "Closed" && openTime <= timestamp && timestamp <= endTime {
					library := make(map[string]interface{})
					for _, element := range LibraryFields {
						library[element] = docData[element]
					}
					libraries = append(libraries, library)
					break
				}
			}
		}
	}
	return libraries, nil
}
