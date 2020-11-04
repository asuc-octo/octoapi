// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "io"
    "net/http"
    "context"
    "bytes"
    "encoding/json"
    "google.golang.org/api/iterator"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
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
type Gyms []Gym

// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func GymEndpoint(w http.ResponseWriter, r *http.Request) {
    client, ctx := initFirestore(w);
    allGyms := getAllGyms(w, client, ctx)
    output, err := json.Marshal(&allGyms)
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
func getAllGyms(w http.ResponseWriter, client *firestore.Client, ctx context.Context) Gyms{
    /* Read Documents from Firestore*/
    defer client.Close()
    var gyms Gyms
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
        gyms = append(gyms, gym)
    }
    //fmt.Fprintf("len=%d cap=%d %v\n", len(gyms), cap(gyms), gyms)
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

