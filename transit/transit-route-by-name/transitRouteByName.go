package transitroutebyname

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var apiURL = "http://api.actransit.org/transit/route"

func TransitRouteByName(w http.ResponseWriter, r *http.Request) {
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

	// Get Route name from Query Param
	route, ok := r.URL.Query()["route"]

	if !ok || len(route[0]) < 1 {
		http.Error(w, "Url Param 'route' is of incorrect type", http.StatusBadRequest)
		return
	}
	// Read Transit API Key from Secrets Manager
	key, err := getTransitSecret(w)
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	// Call Transit API to obtain all routes
	routes, err := getRouteByName(key, route[0])
	if err != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
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
