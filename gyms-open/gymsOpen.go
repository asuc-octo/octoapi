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
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
    "github.com/gorilla/schema"
)
type Timing struct {
    Open_Time int64 `json:"open_time"`
    Close_Time int64 `json:"close_time"`
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

var decoder = schema.NewDecoder()
// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func GymOpenEndpoint(w http.ResponseWriter, r *http.Request) {
    var input struct {
        Time int64 `json:"time"`
    }
    err := decoder.Decode(&input, r.URL.Query())
    if err != nil {
        // Handle error
    }
    client, ctx := initFirestore(w)

    // Distance range
    var timestamp int64
    if input.Time == 0 {
        timestamp = time.Now().Unix()
    } else  {
        timestamp = input.Time
    }
    gyms := getGymsOpen(w, client, ctx, timestamp)
    output, err := json.Marshal(&gyms)
    if err != nil {
        return
    }
    fmt.Fprint(w, string(output))
    	

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


    /* Load Firestore */
    opt := option.WithCredentialsJSON(json_input)
    client, new_err := firestore.NewClient(ctx, "berkeley-mobile", opt)
    if new_err != nil {
        fmt.Fprint(w, "client failed\n")
    }
    return client, ctx
} 

// radius in meters
func getGymsOpen(w http.ResponseWriter, client *firestore.Client, ctx context.Context, timestamp int64) []Gym{
    defer client.Close()
    var gyms = []Gym{}
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
        var gym Gym
        if err := doc.DataTo(&gym); err != nil  {
            fmt.Println(err)
        }
        for _, timing := range gym.Open_Close_Hours {
            if (timing.Open_Time <= timestamp && timestamp <= timing.Close_Time) {
                gyms = append(gyms, gym)
                break
            }
        } 
    }
    return gyms
}

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
