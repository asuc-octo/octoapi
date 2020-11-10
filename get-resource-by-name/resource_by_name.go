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
    "github.com/gorilla/schema"
)

var client *firestore.Client
var ctx context.Context

func ResourceByName(w http.ResponseWriter, r *http.Request) {
	initFirestore(w)

    name, ok := r.URL.Query()["name"]
    
    if !ok || len(name[0]) < 1 {
        http.Error(w, "Url Param 'key' is missing", http.StatusBadRequest)
        return
    }

    jsonString, err := json.Marshal(getResourceByName(w, name[0]))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

	fmt.Fprint(w, string(jsonString))
	
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

func getResourceByName(w http.ResponseWriter, name string) []map[string]interface{} {

	defer client.Close()
	var resources []map[string]interface{}

	iter := client.Collection("Campus Resource").Where("name", "==", name).Documents(ctx)

	for {
		doc, err := iter.Next()
		
		if err == iterator.Done {
                break
        }
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return nil
        }
        resources = append(resources, doc.Data())
	}

	return resources
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