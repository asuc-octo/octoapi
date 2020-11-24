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
    "log"
    "strconv"
    "time"
)

var client *firestore.Client
var ctx context.Context
var decoder = schema.NewDecoder()

func ResourceByTime(w http.ResponseWriter, r *http.Request) {
	err := initFirestore()
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        return
    }

    timeParam, ok := r.URL.Query()["time"]
    var timestamp int64
    if !ok || len(timeParam[0]) < 1 {
        timestamp, err = strconv.ParseInt(timeParam[0], 10, 64)
        if err != nil {
            http.Error(w, "Query param 'time' is of inccorect type", http.StatusInternalServerError)
            return
        }
    } else {
        timestamp = time.Now().Unix()
    }
    
    resources, err := getResourceByTime(timestamp)
    jsonString, err := json.Marshal(resources)
    if err != nil {
        http.Error(w, "Error converting resources to string", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonString))
}

func getResourceByTime(timestamp int64) ([]map[string]interface{}, error) {

	defer client.Close()
    var resources []map[string]interface{}
    floatTimestamp := float64(timestamp)

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()
		
		if err == iterator.Done {
                break
        }
        if err != nil {
                return nil, err
        }
        docData := doc.Data()
        timeArray := docData["open_close_array"].([]interface{})
        for i := 0; i < len(timeArray); i++ {
            entry := timeArray[i].(map[string]interface{})
            openTime := entry["open_time"].(float64)
            closeTime := entry["close_time"].(float64)
            if floatTimestamp < openTime || floatTimestamp > closeTime {
                continue
            }
        }


        resources = append(resources, docData)
	}
	return resources, nil
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

func StreamToByte(stream io.Reader) []byte {
  buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}