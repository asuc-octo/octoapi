// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "net/http"
    "context"
    "encoding/json"
    "errors"
    "strconv"
    "log"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "google.golang.org/api/option"
    "github.com/umahmood/haversine"
    "github.com/martinlindhe/unit"
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
var unitMap = map[string]unit.Length{
    "ft": unit.Foot,
    "yd": unit.Yard,
    "mi": unit.Mile,
    "m": unit.Meter,
    "km": unit.Kilometer,
}
var firestoreKeyResourceID = "projects/980046983693/secrets/firestore_access_key/versions/1"

func GymLocationsEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    var radius float64
    var longitude float64
    var latitude float64
    var units string
    var err error
    radiusInput, ok := r.URL.Query()["radius"]
    if ok {
        if len(radiusInput[0]) >= 1 {
            radius, err = strconv.ParseFloat(radiusInput[0], 64)
            if err != nil {
                http.Error(w, "Url Param 'radius' is of incorrect type", http.StatusBadRequest)
                return
            }
        } else if len(radiusInput[0]) < 1 {
            http.Error(w, "Url Param 'radius' is of incorrect type", http.StatusBadRequest)
            return
        }
    } else {
        http.Error(w, "Url Param 'radius' is missing", http.StatusBadRequest)
        return
    }
    longitudeInput, ok := r.URL.Query()["longitude"]
    if ok {
        if len(longitudeInput[0]) >= 1 {
            longitude, err = strconv.ParseFloat(longitudeInput[0], 64)
            if err != nil {
                http.Error(w, "Url Param 'longitude' is of incorrect type", http.StatusBadRequest)
                return
            }
        } else if len(longitudeInput[0]) < 1 {
            http.Error(w, "Url Param 'longitude' is of incorrect type", http.StatusBadRequest)
            return
        }
    } else {
        http.Error(w, "Url Param 'longitude' is missing", http.StatusBadRequest)
        return
    }

    latitudeInput, ok := r.URL.Query()["latitude"]
    if ok {
        if len(latitudeInput[0]) >= 1 {
            latitude, err = strconv.ParseFloat(latitudeInput[0], 64)
            if err != nil {
                http.Error(w, "Url Param 'latitude' is of incorrect type", http.StatusBadRequest)
                return
            }
        } else if len(latitudeInput[0]) < 1 {
            http.Error(w, "Url Param 'latitude' is of incorrect type", http.StatusBadRequest)
            return
        }
    } else {
        http.Error(w, "Url Param 'latitude' is missing", http.StatusBadRequest)
        return
    }

    unitsInput, ok := r.URL.Query()["unit"]
    if !ok || len(unitsInput) < 1 {
        http.Error(w, "Url Param 'unit' is missing", http.StatusBadRequest)
        return
    }
    units = unitsInput[0]

    var kilometers float64
    kilometers, err = convertToKilometers(radius, units)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
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
    gyms, err = getGymsInRadius(w, longitude, latitude, kilometers)
    if err != nil {
        http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
        log.Printf("Get Gyms in Radius failed: %v", err)
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
func getGymsInRadius(w http.ResponseWriter, longitude float64, latitude float64, radius float64) ([]map[string]interface{}, error){
    defer client.Close()
    gyms := make([]map[string]interface{}, 0)
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
        gymLatitude, okLatitude := gym["latitude"].(float64)
        gymLongitude, okLongitude := gym["longitude"].(float64)
        if !okLatitude {
            log.Printf("Latitude cannot be parsed: %v", gym)
        } else if !okLongitude {
            log.Printf("Longitude cannot be parsed: %v", gym)
        } else {
            _, km := haversine.Distance(haversine.Coord{Lat: latitude, Lon: longitude}, haversine.Coord{Lat: gymLatitude, Lon: gymLongitude})
            if (km <= radius) {
                gyms = append(gyms, gym)
            }
        }
    }
    return gyms, nil
}

func convertToKilometers(value float64, units string) (float64, error) {
    unitConvert, valid := unitMap[units];
    if valid {
        return (unit.Length(value) * unitConvert).Kilometers(), nil
    } else {
        return 0, errors.New("URL Param 'unit' is incorrect")
    }
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
