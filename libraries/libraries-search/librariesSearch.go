package librariessearch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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

func LibrarySearchEndpoint(w http.ResponseWriter, r *http.Request) {
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

	name := strings.ToLower(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "Url Param 'name' is missing", http.StatusBadRequest)
		return
	}
	fstoreErr := initFirestore(w)
	if fstoreErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", fstoreErr)
		return
	}
	libraries, libraryErr := searchLibraries(ctx, w, client, name)
	if libraryErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("libraries search GET failed: %v", libraryErr)
		return
	}
	output, jsonErr := json.Marshal(&libraries)
	if jsonErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("libraries JSON conversion failed: %v", jsonErr)
		return
	}
	fmt.Fprint(w, string(output))
}

func searchLibraries(ctx context.Context, w http.ResponseWriter, client *firestore.Client, name string) ([]map[string]interface{}, error) {
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
		if strings.Contains(strings.ToLower(fmt.Sprintf("%s", docData["name"])), name) {
			library := make(map[string]interface{})
			for _, element := range LibraryFields {
				library[element] = docData[element]
			}
			libraries = append(libraries, library)
		}
	}
	return libraries, nil
}
