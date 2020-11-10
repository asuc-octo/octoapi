// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "io"
    "net/http"
    "context"
    "bytes"
    "encoding/json"
    "time"
    "strconv"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
    "github.com/gorilla/schema"
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

var decoder = schema.NewDecoder()
var client *firestore.Client
var ctx context.Context
var GymFields = [...]string{"name", "description", "latitude", "longitude", "address", "phone", "open_close_hours", "track_hours", "pool_hours"}

func GymOpenEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    var timestamp int64
    var err error
    timestamp = time.Now().Unix()
    timeInput, ok := r.URL.Query()["time"]
    if ok {
        if len(timeInput[0]) >= 1 {
            timestamp, err = strconv.ParseInt(timeInput[0], 10, 64)
            if err != nil {
                http.Error(w, "Url Param 'time' is of incorrect type", http.StatusBadRequest)
                return
            }
        } else if len(timeInput[0]) < 1 {
            http.Error(w, "Url Param 'time' is of incorrect type", http.StatusBadRequest)
            return
        }
    }
    err = initFirestore(w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // Distance range
    var output []byte
    var gyms []map[string]interface{}
    gyms, err = getGymsOpen(w, timestamp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    output, err = json.Marshal(gyms)
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

// radius in meters
func getGymsOpen(w http.ResponseWriter, timestamp int64) ([]map[string]interface{}, error) {
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
        hours, ok := gym["open_close_hours"].([]map[string]int64)
        if !ok {
            fmt.Println("Couldn't parse open_close_hours array")
        } else {
            for _, timing := range hours {
                if (timing["open_time"] <= timestamp && timestamp <= timing["close_time"]) {
                    gyms = append(gyms, gym)
                    break
                }
            } 
        }
    }
    return gyms, nil
}

func StreamToByte(stream io.Reader) []byte {
  buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

