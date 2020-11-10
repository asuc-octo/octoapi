package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var LibraryFields = [6]string{"name", "description", "latitude", "longitude", "address", "open_close_array"}

func main() {
	http.HandleFunc("/", LibrarySearchEndpoint)

	log.Println("STARTED")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func LibrarySearchEndpoint(w http.ResponseWriter, r *http.Request) {
	client, ctx := initFirestore(w)
	name := strings.ToLower(r.URL.Query().Get("name"))
	libraries := searchLibraries(ctx, w, client, name)
	output, err := json.Marshal(&libraries)
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

func searchLibraries(ctx context.Context, w http.ResponseWriter, client *firestore.Client, name string) []map[string]interface{} {
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
		if strings.Contains(strings.ToLower(fmt.Sprintf("%s", docData["name"])), name) {
			library := make(map[string]interface{})
			for _, element := range LibraryFields {
				library[element] = docData[element]
			}
			libraries = append(libraries, library)
		}
	}
	return libraries
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
