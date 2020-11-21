// Package p contains an HTTP Cloud Function.
package p

import (
    "context"
    "fmt"
    "net/http"
    "encoding/json"
    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"

)

var apiURL = "http://api.actransit.org/transit/routes"
var transitKeyResourceID = "projects/980046983693/secrets/transit_api_key/versions/1"

func AllRoutes(w http.ResponseWriter, r *http.Request) {

    // Read Transit API Key from Secrets Manager
    key, err := getTransitSecret(w)
    if err != nil {
        http.Error(w, "Error retrieving api key: " + err.Error(), http.StatusInternalServerError)
        return
    }

    // Call Transit API to obtain all routes
    routes, err := getAllRoutes(key)
    if err != nil {
        http.Error(w, "error retrieving routes" + err.Error(), http.StatusInternalServerError)
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

func getTransitSecret(w http.ResponseWriter) (string, error) {
    ctx := context.Background()
    client, err := secretmanager.NewClient(ctx)
    if err != nil {
        return "", err
    }

    // Build the request.
    req := &secretmanagerpb.AccessSecretVersionRequest{
            Name: transitKeyResourceID,
    }

    // Call the API.
    result, err := client.AccessSecretVersion(ctx, req)
    if err != nil {
            return "", err
    }
    return string(result.Payload.Data), nil
}