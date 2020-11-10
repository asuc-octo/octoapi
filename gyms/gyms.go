// Package p contains an HTTP Cloud Function.
package gyms

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
)
// type Timing struct {
//     Open_Time int64 `json:"open_time"`
//     Close_Time int64 `json:"close_time"`
// }

// type Gym struct {
//     Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Phone string `json:"phone"`
//     Description string `json:"description"`
//     Open_Close_Hours []Timing `json:"open_close_hours"`
//     Track_Hours []Timing `json:"track_hours"`
//     Pool_Hours []Timing `json:"pool_hours"`
// }
var client *firestore.Client
var ctx context.Context
var GymFields = [...]string{"name", "description", "latitude", "longitude", "address", "phone", "open_close_hours", "track_hours", "pool_hours"}

func GymEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    err := initFirestore(w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    var output []byte
    var allGyms []map[string]interface{}
    allGyms, err = getAllGyms(w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    output, err = json.Marshal(allGyms)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Fprint(w, string(output))
}
func initFirestore(w http.ResponseWriter)  error {
    ctx = context.Background()

    /* Get Auth for accessing Firestore by getting json file in cloud storage*/
    storageClient, storageError := storage.NewClient(ctx)
    if storageError != nil {
        fmt.Fprint(w, "storage client failed\n")
        return storageError
    }
    defer storageClient.Close()
    bkt := storageClient.Bucket("firestore_access")
    obj := bkt.Object("berkeley-mobile-e0922919475f.json")
    read, readerError := obj.NewReader(ctx)
    if readerError != nil {
        fmt.Fprint(w, "Reader failed!\n")
        return readerError
    }
    defer read.Close()
    json_input := StreamToByte(read) // the byte array of the json file

    /* Load Firestore */
    var clientErr error
    opt := option.WithCredentialsJSON(json_input)
    client, clientErr = firestore.NewClient(ctx, "berkeley-mobile", opt)
    if clientErr != nil {
        fmt.Fprint(w, "client failed\n")
        return clientErr
    }
    return nil
} 
func getAllGyms(w http.ResponseWriter) ([]map[string]interface{}, error) {
    /* Read Documents from Firestore*/
    defer client.Close()
    var gyms []map[string]interface{}
    iter := client.Collection("Gyms").Documents(ctx)
    for {
        doc, err := iter.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            fmt.Println(err)
            return nil, err
        }
        docData := doc.Data()
        gym := make(map[string]interface{})
        for _, element := range GymFields {
            gym[element] = docData[element]
        }
        gyms = append(gyms, gym)
    }
    return gyms, nil
}

func StreamToByte(stream io.Reader) []byte {
    buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

