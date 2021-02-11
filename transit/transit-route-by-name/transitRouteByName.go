package transitroutebyname

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var apiURL = "http://api.actransit.org/transit/route"

func TransitRouteByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	// Get Route name from Query Param
	route, ok := r.URL.Query()["route"]

	if !ok || len(route[0]) < 1 {
		http.Error(w, "Query Param 'route' is missing", http.StatusBadRequest)
		return
	}
	// Read Transit API Key from Secrets Manager
	key, err := getTransitSecret(w)
	if err != nil {
		http.Error(w, "Error retrieving api key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Call Transit API to obtain all routes
	routes, err := getRouteByName(key, route[0])
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

func getRouteByName(key, route string) (map[string]interface{}, error) {
	requestURL := apiURL + "/" + route + "?token=" + key

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
