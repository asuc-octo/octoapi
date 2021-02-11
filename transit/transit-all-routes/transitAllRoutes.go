package transitallroutes

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var apiURL = "http://api.actransit.org/transit/routes"

func TransitAllRoutesEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	// Read Transit API Key from Secrets Manager
	key, err := getTransitSecret(w)
	if err != nil {
		http.Error(w, "Error retrieving api key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Call Transit API to obtain all routes
	routes, err := getAllRoutes(key)
	if err != nil {
		http.Error(w, "error retrieving routes"+err.Error(), http.StatusInternalServerError)
		return
	}

	// Format results to JSON
	jsonString, err := json.Marshal(routes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func getAllRoutes(key string) ([]map[string]interface{}, error) {
	requestURL := apiURL + "?token=" + key

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var routes []map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&routes)
	if err != nil {
		return nil, err
	}
	return routes, nil
}
