// Package p contains an HTTP Cloud Function.
package gyms

import (
    "fmt"
    "io"
    "html"
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
    Latitude float32 `json:"latitude"`
    Longitude float32 `json:"longitude"`
    Phone string `json:"phone"`
    Description string `json:"description"`
    Open_Close_Hours []Timing `json:"open_close_hours"`
    Track_Hours []Timing `json:"track_hours"`
    Pool_Hours []Timing `json:"pool_hours"`
}

// HelloWorld prints the JSON encoded "message" field in the body
// of the request or "Hello, World!" if there isn't one.
func HelloWorld(w http.ResponseWriter, r *http.Request) {
    var input struct {
        Name string `json:"name"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        return
    }

    if input.Name == "" {
        fmt.Fprint(w, "No input")
        return
    }
    gym := getGym(w, html.EscapeString(input.Name))
    if gym.Name == input.Name {
	   fmt.Fprint(w, gym)
    }
	
    
}
func getAllGyms(w http.ResponseWriter) []Gym{
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
    defer client.Close()
    //fmt.Println(client)

    /* Read Documents from Firestore*/
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

func getGym(w http.ResponseWriter, name string) Gym{
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
    client, new_err := firestore.NewClient(ctx, "berkeley-mobile", opt) //app.Firestore(ctx)
    if new_err != nil {
        fmt.Fprint(w, "client failed\n")
    }
    defer client.Close()

    /* Read Documents from Firestore*/
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

