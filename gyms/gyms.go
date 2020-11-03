// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "io"
    "html"
    "net/http"
    "context"
    "bytes"
    "math"
    "encoding/json"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
    "github.com/umahmood/haversine"
    "github.com/martinlindhe/unit"
)
type Timing struct {
    Open_Time int `json:"open_time"`
    Close_Time int `json:"close_time"`
}

type Gym struct {
    Name string `json:"name"`
    Latitude float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Phone string `json:"phone"`
    Description string `json:"description"`
    Open_Close_Hours []Timing `json:"open_close_hours"`
    Track_Hours []Timing `json:"track_hours"`
    Pool_Hours []Timing `json:"pool_hours"`
}

// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func GymEndpoint(w http.ResponseWriter, r *http.Request) {
    var input struct {
        Name string `json:"name"`
        Longitude float64 `json:"longitude"`
        Latitude float64 `json:"latitude"`
        Radius float64 `json:"radius"`
        Unit string `json:"unit"`
    }
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        return // parameters are wrong
    }
    client, ctx := initFirestore(w);

    if input.Name != "" {
        gym := getGymByName(w, client, ctx, html.EscapeString(input.Name))
        if gym.Name == input.Name {
           fmt.Fprint(w, gym)
        }
    } else {
    
        switch input.Unit {
            case "ft":
                fmt.Fprint(w, (unit.Length(input.Radius) * unit.Foot).Meters())
                fmt.Fprint(w, getGymsInRadius(w, client, ctx, input.Longitude, input.Latitude, (unit.Length(input.Radius) * unit.Foot).Kilometers()))
            case "yd":
                fmt.Fprint(w, (unit.Length(input.Radius) * unit.Yard).Meters())
                fmt.Fprint(w, getGymsInRadius(w, client, ctx, input.Longitude, input.Latitude, (unit.Length(input.Radius) * unit.Yard).Kilometers()))
            case "mi":
                fmt.Fprint(w, (unit.Length(input.Radius) * unit.Mile).Meters())
                fmt.Fprint(w, getGymsInRadius(w, client, ctx, input.Longitude, input.Latitude, (unit.Length(input.Radius) * unit.Mile).Kilometers()))
            case "m":
                fmt.Fprint(w, (unit.Length(input.Radius) * unit.Meter).Meters())
                fmt.Fprint(w, getGymsInRadius(w, client, ctx, input.Longitude, input.Latitude, (unit.Length(input.Radius) * unit.Meter).Kilometers()))
            case "km":
                fmt.Fprint(w, (unit.Length(input.Radius) * unit.Kilometer).Meters())
                fmt.Fprint(w, getGymsInRadius(w, client, ctx, input.Longitude, input.Latitude, (unit.Length(input.Radius) * unit.Kilometer).Kilometers()))
            default:
        }
        
    }
    	

}
func initFirestore(w http.ResponseWriter) (*firestore.Client, context.Context) {
    ctx := context.Background()

    /* Get Auth for accessing Firestore by getting json file in cloud storage*/
    storageClient, err := storage.NewClient(ctx)
    if err != nil {
        fmt.Fprint(w, "storage client failed\n")
    }
    defer storageClient.Close()
    bkt := storageClient.Bucket("firestore_access")
    obj := bkt.Object("berkeley-mobile-e0922919475f.json")
    read, err1 := obj.NewReader(ctx)
    if err1 != nil {
        fmt.Fprint(w, "Reader failed!\n")
    }
    defer read.Close()
    json_input := StreamToByte(read) // the byte array of the json file
    //fmt.Fprint(w, string(json_input))


    /* Load Firestore */
    opt := option.WithCredentialsJSON(json_input)
    //fmt.Fprint(w, "Success Loading Firestore!\n")
    //fmt.Println(opt)
    client, new_err := firestore.NewClient(ctx, "berkeley-mobile", opt) //app.Firestore(ctx)
    if new_err != nil {
        fmt.Fprint(w, "client failed\n")
    }
    //fmt.Println(client)
    return client, ctx
} 
func getAllGyms(w http.ResponseWriter, client *firestore.Client, ctx context.Context) []Gym{
    

    /* Read Documents from Firestore*/
    defer client.Close()
    var gyms []Gym
    iter := client.Collection("Gyms").Documents(ctx)
    for {
        doc, err := iter.Next()
        if err == iterator.Done {
                break
        }
        if err != nil {
                fmt.Println(err)
                return gyms
        }
        //fmt.Println(doc.Data())
        //fmt.Fprintf(w, doc.Data())
        var gym Gym
        if err := doc.DataTo(&gym); err != nil  {
            fmt.Println(err)
        }
        gyms = append(gyms, gym)
    }
    //fmt.Fprintf("len=%d cap=%d %v\n", len(gyms), cap(gyms), gyms)
    return gyms
}

func getGymByName(w http.ResponseWriter, client *firestore.Client, ctx context.Context, name string) Gym{
    
    /* Read Documents from Firestore*/
    defer client.Close()
    var gym Gym
    iter := client.Collection("Gyms").Where("name", "==", name).Documents(ctx)
    doc, err := iter.Next()
    if err == iterator.Done {
        return gym
    }
    if err != nil {
            fmt.Println(err)
            return gym
    }
    
    if err := doc.DataTo(&gym); err != nil  {
        fmt.Println(err)
        return gym
    }
    return gym
}

// radius in meters
func getGymsInRadius(w http.ResponseWriter, client *firestore.Client, ctx context.Context, longitude float64, latitude float64, radius float64) []Gym{
    
    /* Read Documents from Firestore*/
    allGyms := getAllGyms(w, client, ctx)
    var gyms []Gym

    for _, gym := range allGyms {
        _, km := haversine.Distance(haversine.Coord{Lat: latitude, Lon: longitude}, haversine.Coord{Lat: gym.Latitude, Lon: gym.Longitude})
        //dist := Distance(latitude, longitude, gym.Latitude, gym.Longitude)
        fmt.Printf("%f: %f, %f - %f, %f\n", km , latitude , longitude , gym.Latitude , gym.Longitude)
        if (km <= radius) {
            gyms = append(gyms, gym)
        }
    }
    return gyms
}
/*

func hsin(theta float64) float64 {
    return math.Pow(math.Sin(theta/2), 2)
}
func Distance(lat1, lon1, lat2, lon2 float64) float64 {
    // convert to radians
  // must cast radius as float to multiply later
    var la1, lo1, la2, lo2, r float64
    la1 = lat1 * math.Pi / 180
    lo1 = lon1 * math.Pi / 180
    la2 = lat2 * math.Pi / 180
    lo2 = lon2 * math.Pi / 180

    r = 6378100 // Earth radius in METERS

    // calculate
    h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

    return 2 * r * math.Asin(math.Sqrt(h))
}
*/
func StreamToByte(stream io.Reader) []byte {
  buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}

