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
    "github.com/gorilla/schema"
    "strconv"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()

func ResourceByTime(w http.ResponseWriter, r *http.Request) {
    initFirestore(w)

    timeParam, ok := r.URL.Query()["time"]
    if !ok || len(timeParam[0]) < 1 {
        http.Error(w, "Url Param 'time' is missing", http.StatusBadRequest)
        return
    }

    timestamp, err := strconv.ParseFloat(timeParam[0], 64)
    if err != nil {
        http.Error(w, "Couldn't parse timestamp", http.StatusInternalServerError)
        return
    }
    

    jsonString, err := json.Marshal(getResourceByTime(w, timestamp))
    if err != nil {
        http.Error(w, "Error converting resources to string", http.StatusInternalServerError)
        return
    }

	fmt.Fprint(w, string(jsonString))
	
}

func getResourceByTime(w http.ResponseWriter, timestamp float64) []map[string]interface{} {

	defer client.Close()
	var resources []map[string]interface{}

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()
		
		if err == iterator.Done {
                break
        }
        if err != nil {
                http.Error(w, "getResourceByTime: " + err.Error(), http.StatusInternalServerError)
                return nil
        }
        docData := doc.Data()
        timeArray := docData["open_close_array"].([]interface{})
        for i := 0; i < len(timeArray); i++ {
            entry := timeArray[i].(map[string]interface{})
            openTime := entry["open_time"].(float64)
            closeTime := entry["close_time"].(float64)
            if timestamp < openTime || timestamp > closeTime {
                continue
            }
        }


        resources = append(resources, docData)
	}
	return resources
}

func initFirestore(w http.ResponseWriter) {
	ctx = context.Background()

	storageClient, err := storage.NewClient(ctx)
    if err != nil {
        http.Error(w, "initFirestore: " + err.Error(), http.StatusInternalServerError)
    }
    defer storageClient.Close()
    bkt := storageClient.Bucket("firestore_access")
    obj := bkt.Object("berkeley-mobile-e0922919475f.json")
    read, readerErr := obj.NewReader(ctx)
    if readerErr != nil {
        http.Error(w, "initFirestore: " + readerErr.Error(), http.StatusInternalServerError)
    }
    defer read.Close()
    json_input := StreamToByte(read)

    opt := option.WithCredentialsJSON(json_input)
	var clientErr error
    client, clientErr = firestore.NewClient(ctx, "berkeley-mobile", opt) //app.Firestore(ctx)
    if clientErr != nil {
        http.Error(w, "initFirestore: " + clientErr.Error(), http.StatusInternalServerError)
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