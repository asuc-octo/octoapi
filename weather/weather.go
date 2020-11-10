// Package p contains an HTTP Cloud Function.
package weather

import (
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "context"
    "bytes"
    "cloud.google.com/go/storage"
)

// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func WeatherEndpoint(w http.ResponseWriter, r *http.Request) {
    apiKey, err := getAPIKey(w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    resp, err := http.Get("https://api.openweathermap.org/data/2.5/onecall?lat=37.8712&lon=-122.2601&appid=" + apiKey)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, string(body))
}

func getAPIKey(w http.ResponseWriter) (string, error) {
    ctx := context.Background()
    storageClient, err := storage.NewClient(ctx)
    if err != nil {
        fmt.Fprint(w, "storage client failed\n")
        return "", err
    }
    defer storageClient.Close()
    bkt := storageClient.Bucket("weather_api_access")
    obj := bkt.Object("weather_map_api_key.txt")
    read, err1 := obj.NewReader(ctx)
    if err1 != nil {
        fmt.Fprint(w, "Reader failed!\n")
        return "", err1
    }
    defer read.Close()
    apiKey := StreamToString(read)
    return apiKey, nil
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}