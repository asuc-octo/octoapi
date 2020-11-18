// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "html"
    "net/http"
    "context"
    "log"
    "encoding/json"
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


func GymSearchEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    name, ok := r.URL.Query()["name"]
    if !ok || len(name[0]) < 1 {
        http.Error(w, "Url Param 'name' is missing", http.StatusBadRequest)
        return
    }
    err := initFirestore(w)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Firestore Init failed: %v", err)
        return
    }
    // Search by name
    var output []byte
    var gym map[string]interface{}
    gym, err = getGymByName(w, html.EscapeString(name[0]))
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Get Name failed: %v", err)
        return
    }
    output, err = json.Marshal(&gym)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Couldn't convert gym to JSON: %v", err)
        return
    }
    fmt.Fprint(w, string(output))
    return
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
func getGymByName(w http.ResponseWriter, name string) (map[string]interface{}, error){
    /* Read Documents from Firestore*/
    defer client.Close()
    var gym map[string]interface{}    
    iter := client.Collection("Gyms").Where("name", "==", name).Documents(ctx)
    doc, err := iter.Next()
    if err == iterator.Done {
        return nil, nil
    }
    if err != nil {
        fmt.Println(err)
        return nil, err
    }
    
    docData := doc.Data()
    gym = make(map[string]interface{})
    for _, element := range GymFields {
        gym[element] = docData[element]
    }
    return gym, nil
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
