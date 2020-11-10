package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect" // add

	"cloud.google.com/go/firestore"
	"github.com/gorilla/schema"     // add
	"github.com/martinlindhe/unit"  // add
	"github.com/umahmood/haversine" // add
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var DiningFields = [7]string{"name", "description", "latitude", "longitude", "address", "picture", "phone"}
var decoder = schema.NewDecoder()
var floatType = reflect.TypeOf(float64(0))

func main() {
	http.HandleFunc("/", DiningLocationEndpoint)

	log.Println("STARTED")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func DiningLocationEndpoint(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
		Radius    float64 `json:"radius"`
		Unit      string  `json:"unit"`
	}
	err := decoder.Decode(&input, r.URL.Query())
	if err != nil {
		// Handle error
	}

	client, ctx := initFirestore(w)
	dinings := locateDinings(ctx, w, client, input.Longitude, input.Latitude, convertToKilometers(input.Radius, input.Unit))
	output, err := json.Marshal(&dinings)
	if err != nil {
		return
	}
	fmt.Fprint(w, string(output))
}

func initFirestore(w http.ResponseWriter) (*firestore.Client, context.Context) {
	ctx := context.Background()
	opt := option.WithCredentialsFile("../../berkeley-mobile-e0922919475f.json")
	client, clientErr := firestore.NewClient(ctx, "berkeley-mobile", opt)
	if clientErr != nil {
		fmt.Fprint(w, "client failed\n")
	}
	return client, ctx
}

func locateDinings(ctx context.Context, w http.ResponseWriter, client *firestore.Client, longitude float64, latitude float64, radius float64) []map[string]interface{} {
	defer client.Close()
	var dinings []map[string]interface{}
	iter := client.Collection("Dining Halls").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			return dinings
		}
		docData := doc.Data()
		docLat, latErr := getFloat(docData["latitude"])
		docLon, lonErr := getFloat(docData["longitude"])
		if latErr != nil || lonErr != nil {
			fmt.Println(err)
			return dinings
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
	return dinings
}

func convertToKilometers(value float64, units string) float64 {
	switch units {
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
	return 0.0
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
