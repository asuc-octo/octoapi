// Package p contains an HTTP Cloud Function.
package p

import (
    "fmt"
    "io"
    "net/http"
    "context"
    "bytes"
    "encoding/json"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
    "github.com/martinlindhe/unit"
    "github.com/umahmood/haversine"
    "github.com/gorilla/schema"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()

func ResourceByRange(w http.ResponseWriter, r *http.Request) {
    initFirestore(w)

    var input struct {
        Longitude float64 `json:"longitude"`
        Latitude float64 `json:"latitude"`
        Radius float64 `json:"radius"`
        Unit string `json:"unit"`
    }
    err := decoder.Decode(&input, r.URL.Query())
    if err != nil || input.Longitude == 0 || input.Latitude == 0 || input.Radius == 0 || input.Unit == "" {
        http.Error(w, "Missing parameters or incorrect types", http.StatusBadRequest)
        return
    }

    jsonString, err := json.Marshal(getResourceByRange(w, input.Longitude, input.Latitude, convertToKilometers(input.Radius, input.Unit)))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

	fmt.Fprint(w, string(jsonString))
	
}

func getResourceByRange(w http.ResponseWriter, longitude float64, latitude float64, radius float64) []map[string]interface{} {

	defer client.Close()
	var resources []map[string]interface{}

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()
		
		if err == iterator.Done {
                break
        }
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return nil
        }

        docData := doc.Data()
        docLat, ok := docData["latitude"].(float64)
        docLon, ok:= docData["longitude"].(float64)
        if !ok {
            fmt.Println("Error: Can't entry doesn't have location data available or is not of type float")
            continue
        }

        _, km := haversine.Distance(haversine.Coord{Lat: latitude, Lon: longitude},
            haversine.Coord{Lat: docLat, Lon: docLon})
            
        delete(docData, "latitude")
		if km <= radius {
			resources = append(resources, docData)
		}
	}
	return resources
}

func initFirestore(w http.ResponseWriter) {
	ctx = context.Background()

	storageClient, err := storage.NewClient(ctx)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
    defer storageClient.Close()
    bkt := storageClient.Bucket("firestore_access")
    obj := bkt.Object("berkeley-mobile-e0922919475f.json")
    read, readerErr := obj.NewReader(ctx)
    if readerErr != nil {
        http.Error(w, readerErr.Error(), http.StatusInternalServerError)
    }
    defer read.Close()
    json_input := StreamToByte(read)

    opt := option.WithCredentialsJSON(json_input)
	var clientErr error
    client, clientErr = firestore.NewClient(ctx, "berkeley-mobile", opt) //app.Firestore(ctx)
    if clientErr != nil {
        http.Error(w, clientErr.Error(), http.StatusInternalServerError)
    }
}

func convertToKilometers(value float64, units string) float64 {
    switch units {
        case "ft":
            return (unit.Length(value) * unit.Foot).Kilometers()
        case "yd":
            return (unit.Length(value) * unit.Yard).Kilometers()
        case "mi":
            return (unit.Length(value) * unit.Mile).Kilometers()
        case "m":
            return (unit.Length(value) * unit.Meter).Kilometers()
        case "km":
            return (unit.Length(value) * unit.Kilometer).Kilometers()
    }
    return 0.0
}


func StreamToByte(stream io.Reader) []byte {
  buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}