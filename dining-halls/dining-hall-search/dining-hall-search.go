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

var DiningFields = [7]string{"name", "description", "latitude", "longitude", "address", "picture", "phone"}

func main() {
	http.HandleFunc("/", DiningSearchEndpoint)

	log.Println("STARTED")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func DiningSearchEndpoint(w http.ResponseWriter, r *http.Request) {
	client, ctx := initFirestore(w)
	name := strings.ToLower(r.URL.Query().Get("name"))
	dinings := searchDinings(ctx, w, client, name)
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

func searchDinings(ctx context.Context, w http.ResponseWriter, client *firestore.Client, name string) []map[string]interface{} {
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
		if strings.Contains(strings.ToLower(fmt.Sprintf("%s", docData["name"])), name) {
			dining := make(map[string]interface{})
			for _, element := range DiningFields {
				dining[element] = docData[element]
			}
			dinings = append(dinings, dining)
		}
	}
	return dinings
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
