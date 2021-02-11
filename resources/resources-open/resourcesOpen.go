package resourcesopen

import (
	"context"
	"encoding/json"
	"fmt"
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

	resources, err := getOpenResources(w)
	if err != nil {
		http.Error(w, "Error obtaining resources: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonString, err := json.Marshal(resources)
	if err != nil {
		http.Error(w, "Error converting resources to string: "+err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "getResourceByTime: "+err.Error(), http.StatusInternalServerError)
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
