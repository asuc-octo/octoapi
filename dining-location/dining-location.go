package dining

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/gorilla/schema"
	"github.com/martinlindhe/unit"
	"github.com/umahmood/haversine"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// type Dining struct {
// 	Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Phone string `json:"phone"`
//     Description string `json:"description"`
//     Address string `json:"address"`
// }

var DiningFields = [6]string{"name", "description", "latitude", "longitude", "address", "phone"}
var decoder = schema.NewDecoder()
var floatType = reflect.TypeOf(float64(0))
var unitMap = map[string]unit.Length{
	"ft": unit.Foot,
	"yd": unit.Yard,
	"mi": unit.Mile,
	"m":  unit.Meter,
	"km": unit.Kilometer,
}

func DiningLocationEndpoint(w http.ResponseWriter, r *http.Request) {
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
	kilometers, convertErr := convertToKilometers(radius, units)
	if convertErr != nil {
		http.Error(w, convertErr.Error(), http.StatusBadRequest)
		return
	}
	client, ctx, fstoreErr := initFirestore(w)
	if fstoreErr != nil {
		http.Error(w, fstoreErr.Error(), http.StatusInternalServerError)
		return
	}
	dinings, diningErr := locateDinings(ctx, w, client, longitude, latitude, kilometers)
	if diningErr != nil {
		http.Error(w, diningErr.Error(), http.StatusInternalServerError)
		return
	}
	output, jsonErr := json.Marshal(&dinings)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(output))
}

func initFirestore(w http.ResponseWriter) (*firestore.Client, context.Context, error) {
	ctx := context.Background()

	/* Get Auth for accessing Firestore by getting json file in cloud storage*/
	storageClient, clientErr := storage.NewClient(ctx)
	if clientErr != nil {
		return nil, nil, clientErr
	}
	defer storageClient.Close()
	bkt := storageClient.Bucket("firestore_access")
	obj := bkt.Object("berkeley-mobile-e0922919475f.json")
	read, readErr := obj.NewReader(ctx)
	if readErr != nil {
		return nil, nil, readErr
	}
	defer read.Close()
	json_input := StreamToByte(read) // the byte array of the json file

	/* Load Firestore */
	opt := option.WithCredentialsJSON(json_input)
	client, new_err := firestore.NewClient(ctx, "berkeley-mobile", opt)
	if new_err != nil {
		return nil, nil, new_err
	}
	return client, ctx, nil
}

func locateDinings(ctx context.Context, w http.ResponseWriter, client *firestore.Client, longitude float64, latitude float64, radius float64) ([]map[string]interface{}, error) {
	defer client.Close()
	dinings := make([]map[string]interface{}, 0)
	iter := client.Collection("Dining Halls").Documents(ctx)
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
			dining := make(map[string]interface{})
			for _, element := range DiningFields {
				dining[element] = docData[element]
			}
			dinings = append(dinings, dining)
		}
	}
	return dinings, nil
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

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
