package dining

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// type Dining struct {
// 	Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Phone string `json:"phone"`
//     Description string `json:"description"`
//     Address string `json:"address"`
// }

var DiningFields = [6]string{"name", "description", "latitude", "longitude", "address", "phone"}
var client *firestore.Client
var ctx context.Context

func DiningEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
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

	fstoreErr := initFirestore(w)
	if fstoreErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", fstoreErr)
		return
	}
	dinings, diningErr := listDinings(ctx, w, client)
	if diningErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("dining GET failed: %v", diningErr)
		return
	}
	output, jsonErr := json.Marshal(&dinings)
	if jsonErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("libraries JSON conversion failed: %v", jsonErr)
		return
	}
	fmt.Fprint(w, string(output))
}

func listDinings(ctx context.Context, w http.ResponseWriter, client *firestore.Client) ([]map[string]interface{}, error) {
	defer client.Close()
	var dinings []map[string]interface{}
	iter := client.Collection("Dining Halls").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docData := doc.Data()
		dining := make(map[string]interface{})
		for _, element := range DiningFields {
			dining[element] = docData[element]
		}
		dinings = append(dinings, dining)
	}
	return dinings, nil
}
