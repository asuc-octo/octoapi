package gymssearch

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"

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

func GymSearchEndpoint(w http.ResponseWriter, r *http.Request) {
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

	name, ok := r.URL.Query()["name"]
	if !ok || len(name[0]) < 1 {
		http.Error(w, "Url Param 'name' is missing", http.StatusBadRequest)
		return
	}
	err := initFirestore(w)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", err)
		return
	}
	// Search by name
	var output []byte
	var gym map[string]interface{}
	gym, err = getGymByName(w, html.EscapeString(name[0]))
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Get Name failed: %v", err)
		return
	}
	output, err = json.Marshal(&gym)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Couldn't convert gym to JSON: %v", err)
		return
	}
	fmt.Fprint(w, string(output))
	return
}

func getGymByName(w http.ResponseWriter, name string) (map[string]interface{}, error) {
	/* Read Documents from Firestore*/
	defer client.Close()
	var gym map[string]interface{}
	iter := client.Collection("Gyms").Where("name", "==", name).Documents(ctx)
	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	docData := doc.Data()
	gym = make(map[string]interface{})
	for _, element := range GymFields {
		gym[element] = docData[element]
	}
	return gym, nil
}
