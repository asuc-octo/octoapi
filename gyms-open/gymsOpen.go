// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "net/http"
    "context"
    "encoding/json"
    "log"
    "time"
    "strconv"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "google.golang.org/api/option"
    "github.com/gorilla/schema"
    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
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
var GymFields = [...]string{"name", "description", "latitude", "longitude", "address", "phone", "open_close_array", "track_hours", "pool_hours"}
var firestoreKeyResourceID = "projects/980046983693/secrets/firestore_access_key/versions/1"

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
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Firestore Init failed: %v", err)
        return
    }
    // Distance range
    var output []byte
    var gyms []map[string]interface{}
    gyms, err = getGymsOpen(w, timestamp)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Get Gyms Open failed: %v", err)
        return
    }
    output, err = json.Marshal(gyms)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Couldn't convert gym to JSON: %v", err)
        return
    }
    fmt.Fprint(w, string(output))
}
func initFirestore(w http.ResponseWriter)  error {
    ctx = context.Background()
    /* Get Auth for accessing Firestore by getting firestore secret */
    key, err := getFirestoreSecret(w)
    if err != nil {
        return err
    }
    /* Load Firestore */
    var clientErr error
    opt := option.WithCredentialsJSON([]byte(key))
    client, clientErr = firestore.NewClient(ctx, "berkeley-mobile", opt)
    if clientErr != nil {
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
            return nil, err
        }
        docData := doc.Data()
        gym := make(map[string]interface{})
        for _, element := range GymFields {
            gym[element] = docData[element]
        }
        hours, ok := gym["open_close_array"].([]map[string]interface{})
        if !ok {
            log.Printf("Couldn't parse open_close_array: %v", gym)
        } else {
            for _, timing := range hours {
                var open_time int64
                open_time, ok = timing["open_time"].(int64)
                if !ok {
                    log.Printf("Couldn't parse open_time: %v", gym)
                } else {
                    var close_time int64
                    close_time, ok = timing["close_time"].(int64)
                    if !ok {
                        log.Printf("Couldn't parse close_time: %v", gym)
                    } else {
                        if (open_time <= timestamp && timestamp <= close_time) {
                            gyms = append(gyms, gym)
                            break
                        }
                    }
                }
            } 
        }
    }
    return gyms, nil
}
func getFirestoreSecret(w http.ResponseWriter) (string, error) {
    ctx := context.Background()
    client, err := secretmanager.NewClient(ctx)
    if err != nil {
        return "", err
    }
    // Build the request.
    req := &secretmanagerpb.AccessSecretVersionRequest{
            Name: firestoreKeyResourceID,
    }
    // Call the API.
    result, err := client.AccessSecretVersion(ctx, req)
    if err != nil {
            return "", err
    }
    return string(result.Payload.Data), nil
}

