package resourcesopen

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/schema"
	"google.golang.org/api/iterator"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()

func ResourcesOpenEndpoint(w http.ResponseWriter, r *http.Request) {
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

	err := initFirestore(w)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	resources, err := getOpenResources(w)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("libraries search GET failed: %v", err)
		return
	}
	jsonString, err := json.Marshal(resources)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("libraries json convert failed: %v", err)
		return
	}
	fmt.Fprint(w, string(jsonString))
}

func getOpenResources(w http.ResponseWriter) ([]map[string]interface{}, error) {

	defer client.Close()
	var resources []map[string]interface{}

	timestamp := time.Now().Unix()

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
			log.Printf("resources fetch data failed %v", err)
			return nil, err
		}
		docData := doc.Data()
		timeArray := docData["open_close_array"].([]interface{})
		for i := 0; i < len(timeArray); i++ {
			entry := timeArray[i].(map[string]interface{})
			openTime := int64(entry["open_time"].(float64))
			closeTime := int64(entry["close_time"].(float64))

			if timestamp < openTime || timestamp > closeTime {
				continue
			}
		}

		resources = append(resources, docData)
	}
	return resources, nil
}
