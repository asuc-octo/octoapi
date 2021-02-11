package transitroutebystop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var apiURL = "http://api.actransit.org/transit/stop/{stopId}/destinations"

func TransitRouteByStopEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	// Get Route name from Query Param
	stopID, ok := r.URL.Query()["stopID"]

	if !ok || len(stopID[0]) < 1 {
		http.Error(w, "Error parsing query params: ", http.StatusBadRequest)
		return
	}
	// Read Transit API Key from Secrets Manager
	key, err := getTransitSecret(w)
	if err != nil {
		http.Error(w, "Error retrieving api key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Call Transit API to obtain all routes
	routes, err := getRoutesByStop(w, key, stopID[0])
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

func getRoutesByStop(w http.ResponseWriter, key, stopID string) (map[string]interface{}, error) {
	requestURL := strings.Replace(apiURL, "{stopId}", stopID, 1)
	requestURL = requestURL + "?token=" + key

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var routes map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&routes)
	if err != nil {
		return nil, err
	}
	return routes, nil
}
