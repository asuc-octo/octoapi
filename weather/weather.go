// Package p contains an HTTP Cloud Function.
package weather

import (
    "fmt"
    "log"
    "io/ioutil"
    "net/http"
    "context"
    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)
var weatherKeyResourceID = "projects/980046983693/secrets/weather_access_key/versions/1"

// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func WeatherEndpoint(w http.ResponseWriter, r *http.Request) {
    apiKey, err := getWeatherSecret(w)
    if err != nil {
        http.Error(w, "Couldn't connect to Weather API", http.StatusInternalServerError)
        log.Printf("Weather Secret loading failed: %v", err)
        return
    }
    resp, err := http.Get("https://api.openweathermap.org/data/2.5/onecall?lat=37.8712&lon=-122.2601&appid=" + apiKey)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        log.Printf("Weather API error: %v", err)
        return
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, "Couldn't connect to Weather API", http.StatusInternalServerError)
        log.Printf("Couldn't convert Weather API output to our output: %v", err)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, string(body))
}

func getWeatherSecret(w http.ResponseWriter) (string, error) {
    ctx := context.Background()
    client, err := secretmanager.NewClient(ctx)
    if err != nil {
        return "", err
    }

    // Build the request.
    req := &secretmanagerpb.AccessSecretVersionRequest{
            Name: weatherKeyResourceID,
    }

    // Call the API.
    result, err := client.AccessSecretVersion(ctx, req)
    if err != nil {
            return "", err
    }
    return string(result.Payload.Data), nil
}