package libraries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/martinlindhe/unit"
	"github.com/umahmood/haversine"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// type Timing struct {
//     Open_Time int64 `json:"open_time"`
//     Close_Time int64 `json:"close_time"`
// }

// type Library struct {
// 	Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Description string `json:"description"`
//     Address string `json:"address"`
//     Open_Close_Hours []Timing `json:"open_close_hours"`
// }

var LibraryFields = [6]string{"name", "description", "latitude", "longitude", "address", "open_close_array"}
var floatType = reflect.TypeOf(float64(0))
var unitMap = map[string]unit.Length{
	"ft": unit.Foot,
	"yd": unit.Yard,
	"mi": unit.Mile,
	"m":  unit.Meter,
	"km": unit.Kilometer,
}
var client *firestore.Client
var ctx context.Context
var firestoreKeyResourceID = "projects/980046983693/secrets/firestore_access_key/versions/1"

func LibrariesLocationEndpoint(w http.ResponseWriter, r *http.Request) {
	var radius float64
	var longitude float64
	var latitude float64
	var units string
	var err error
	w.Header().Set("Content-Type", "application/json")
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
	kilometers, convertErr := convertToKilometers(radius, units)
	if convertErr != nil {
		http.Error(w, convertErr.Error(), http.StatusBadRequest)
		return
	}
	fstoreErr := initFirestore(w)
	if fstoreErr != nil {
		http.Error(w, fstoreErr.Error(), http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", fstoreErr)
		return
	}
	libraries, libraryErr := locateLibraries(ctx, w, client, longitude, latitude, kilometers)
	if libraryErr != nil {
		http.Error(w, libraryErr.Error(), http.StatusInternalServerError)
		log.Printf("libraries location GET failed: %v", libraryErr)
		return
	}
	output, jsonErr := json.Marshal(&libraries)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
		log.Printf("libraries JSON conversion failed: %v", jsonErr)
		return
	}
	fmt.Fprint(w, string(output))
}

func initFirestore(w http.ResponseWriter) error {
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

func locateLibraries(ctx context.Context, w http.ResponseWriter, client *firestore.Client, longitude float64, latitude float64, radius float64) ([]map[string]interface{}, error) {
	defer client.Close()
	libraries := make([]map[string]interface{}, 0)
	iter := client.Collection("Libraries").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docData := doc.Data()
		docLat, latErr := getFloat(docData["latitude"])
		docLon, lonErr := getFloat(docData["longitude"])
		if latErr != nil || lonErr != nil {
			continue
		}
		_, km := haversine.Distance(haversine.Coord{Lat: latitude, Lon: longitude},
			haversine.Coord{Lat: docLat, Lon: docLon})
		if km < radius {
			library := make(map[string]interface{})
			for _, element := range LibraryFields {
				library[element] = docData[element]
			}
			libraries = append(libraries, library)
		}
	}
	return libraries, nil
}

func convertToKilometers(value float64, units string) (float64, error) {
	unitConvert, valid := unitMap[units]
	if valid {
		return (unit.Length(value) * unitConvert).Kilometers(), nil
	} else {
		return 0, errors.New("URL Param 'unit' is incorrect")
	}
}

func getFloat(unk interface{}) (float64, error) {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return 0, fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
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
