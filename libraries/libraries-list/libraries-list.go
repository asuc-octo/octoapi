package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var LibraryFields = [6]string{"name", "description", "latitude", "longitude", "address", "open_close_array"}

func main() {
	http.HandleFunc("/", LibraryEndpoint)

	log.Println("STARTED")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func LibraryEndpoint(w http.ResponseWriter, r *http.Request) {
	client, ctx := initFirestore(w)
	dinings := listLibraries(ctx, w, client)
	output, err := json.Marshal(&dinings)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
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

func listLibraries(ctx context.Context, w http.ResponseWriter, client *firestore.Client) []map[string]interface{} {
	defer client.Close()
	var libraries []map[string]interface{}
	iter := client.Collection("Libraries").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			return libraries
		}
		docData := doc.Data()
		library := make(map[string]interface{})
		for _, element := range LibraryFields {
			library[element] = docData[element]
		}
		libraries = append(libraries, library)
	}
	return libraries
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
