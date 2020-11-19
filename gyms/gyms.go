// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "net/http"
    "context"
    "log"
    "encoding/json"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "google.golang.org/api/option"
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
var client *firestore.Client
var ctx context.Context
var GymFields = [...]string{"name", "description", "latitude", "longitude", "address", "phone", "open_close_array", "track_hours", "pool_hours"}
var firestoreKeyResourceID = "projects/980046983693/secrets/firestore_access_key/versions/1"

func GymEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    err := initFirestore(w)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Firestore Init failed: %v", err)
        return
    }
    var output []byte
    var allGyms []map[string]interface{}
    allGyms, err = getAllGyms(w)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Get All Gyms failed: %v", err)
        return
    }
    output, err = json.Marshal(allGyms)
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
func getAllGyms(w http.ResponseWriter) ([]map[string]interface{}, error) {
    /* Read Documents from Firestore*/
    defer client.Close()
    gyms := make([]map[string]interface{}, 0)
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


