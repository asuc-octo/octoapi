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
    "log"
)

var client *firestore.Client
var ctx context.Context

func ResourceByName(w http.ResponseWriter, r *http.Request) {
	err := initFirestore()
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        return
    }

    name, ok := r.URL.Query()["name"]
    
    if !ok || len(name[0]) < 1 {
        http.Error(w, "Url Param 'name' is missing", http.StatusBadRequest)
        return
    }

    resources, err := getResourceByName(name[0])
    if err != nil {
        http.Error(w, "Couldn't obtain resources", http.StatusInternalServerError)
        return
    }

    jsonString, err := json.Marshal(resources)
    if err != nil {
        http.Error(w, "Couldn't convert resources to json format", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonString))
}

func initFirestore() error {
	ctx = context.Background()

	storageClient, err := storage.NewClient(ctx)
    if err != nil {
        log.Println(err.Error())
        return err
    }
    defer storageClient.Close()
    bkt := storageClient.Bucket("firestore_access")
    obj := bkt.Object("berkeley-mobile-e0922919475f.json")
    read, readerErr := obj.NewReader(ctx)
    if readerErr != nil {
        log.Println(readerErr.Error())
        return readerErr
    }
    defer read.Close()
    json_input := StreamToByte(read)

    opt := option.WithCredentialsJSON(json_input)
	var clientErr error
    client, clientErr = firestore.NewClient(ctx, "berkeley-mobile", opt) //app.Firestore(ctx)
    if clientErr != nil {
        log.Println(clientErr.Error())
        return clientErr
    }

    return nil
}

func getResourceByName(name string) ([]map[string]interface{}, error) {

	defer client.Close()
	var resources []map[string]interface{}

	iter := client.Collection("Campus Resource").Where("name", "==", name).Documents(ctx)

	for {
		doc, err := iter.Next()
		
		if err == iterator.Done {
                break
        }
        if err != nil {
                return nil, err
        }
        resources = append(resources, doc.Data())
	}

	return resources, nil
}


func StreamToByte(stream io.Reader) []byte {
  buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}