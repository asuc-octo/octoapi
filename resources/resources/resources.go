package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var client *firestore.Client
var ctx context.Context

func CampusResourceEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	err := initFirestore(w)
	if err != nil {
		http.Error(w, "Couldn't connect to the database", http.StatusInternalServerError)
		return
	}

	jsonString, err := json.Marshal(getAllResources(w))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func getAllResources(w http.ResponseWriter) []map[string]interface{} {

	defer client.Close()
	var resources []map[string]interface{}

	iter := client.Collection("Campus Resource").Documents(ctx)

	for {
		doc, err := iter.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}

		resources = append(resources, doc.Data())
	}

	return resources
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
