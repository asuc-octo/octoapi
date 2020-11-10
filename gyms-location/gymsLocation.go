// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "io"
    "net/http"
    "context"
    "bytes"
    "encoding/json"
    "errors"
    "strconv"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
    "github.com/umahmood/haversine"
    "github.com/martinlindhe/unit"
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
var unitMap = map[string]unit.Length{
    "ft": unit.Foot,
    "yd": unit.Yard,
    "mi": unit.Mile,
    "m": unit.Meter,
    "km": unit.Kilometer,
}
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

    // Distance range
    //fmt.Fprint(w, convertToKilometers(input.Radius, input.Unit))
    

    err = initFirestore(w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // Distance range
    var output []byte
    var gyms []map[string]interface{}
    gyms, err = getGymsInRadius(w, longitude, latitude, kilometers)
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
func getGymsInRadius(w http.ResponseWriter, longitude float64, latitude float64, radius float64) ([]map[string]interface{}, error){
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
        gymLatitude, okLatitude := gym["latitude"].(float64)
        gymLongitude, okLongitude := gym["longitude"].(float64)
        if !okLatitude {
            fmt.Println("Couldn't parse latitude")
        } else if !okLongitude {
            fmt.Println("Couldn't parse longitude")
        } else {
            _, km := haversine.Distance(haversine.Coord{Lat: latitude, Lon: longitude}, haversine.Coord{Lat: gymLatitude, Lon: gymLongitude})
            fmt.Printf("%f: %f, %f - %f, %f\n", km , latitude , longitude , gymLatitude , gymLongitude)
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
    /*switch units {
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
    return 0.0*/
}

func StreamToByte(stream io.Reader) []byte {
  buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

