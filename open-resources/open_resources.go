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
    "time"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()

func OpenResources(w http.ResponseWriter, r *http.Request) {
    initFirestore(w)


    resources, err := getOpenResources(w)
    if err != nil {
        http.Error(w, "Error obtaining responses: " + err.Error(), http.StatusInternalServerError)
        return
    }
    jsonString, err := json.Marshal(resources)
    if err != nil {
        http.Error(w, "Error converting resources to string: " + err.Error(), http.StatusInternalServerError)
        return
    }

	fmt.Fprint(w, string(jsonString))
	
}

func getOpenResources(w http.ResponseWriter) ([]map[string]interface{}, error) {

	defer client.Close()
    var resources []map[string]interface{}
    
    timestamp := time.Now().Unix()

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()
		
		if err == iterator.Done {
                break
        }
        if err != nil {
                http.Error(w, "getResourceByTime: " + err.Error(), http.StatusInternalServerError)
                return nil, err
        }
        docData := doc.Data()
        timeArray := docData["open_close_array"].([]interface{})
        for i := 0; i < len(timeArray); i++ {
            entry := timeArray[i].(map[string]interface{})
            openTime := int64(entry["open_time"].(float64))
            closeTime := int64(entry["close_time"].(float64))

            if timestamp < openTime || timestamp > closeTime {
                continue
            }
        }


        resources = append(resources, docData)
	}
	return resources, nil
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