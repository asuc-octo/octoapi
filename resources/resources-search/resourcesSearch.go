package resourcessearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var client *firestore.Client
var ctx context.Context

func ResourcesSearchEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenValid := validateAccessToken(r)
	if !tokenValid {
		http.Error(w, "Invalid access token", http.StatusBadRequest)
		return
	}

	err := initFirestore(w)
	if err != nil {
		http.Error(w, "Couldn't connect to database", http.StatusInternalServerError)
		return
	}

	name, ok := r.URL.Query()["name"]

	if !ok || len(name[0]) < 1 {
		http.Error(w, "Url Param 'name' is missing", http.StatusBadRequest)
		return
	}

	resources, err := getResourceByName(name[0])
	if err != nil {
		http.Error(w, "Couldn't obtain resources", http.StatusInternalServerError)
		return
	}

	jsonString, err := json.Marshal(resources)
	if err != nil {
		http.Error(w, "Couldn't convert resources to json format", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func getResourceByName(name string) (map[string]interface{}, error) {

	defer client.Close()
	var resource map[string]interface{}

	iter := client.Collection("Campus Resource").Where("name", "==", name).Documents(ctx)

	for {
		doc, err := iter.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		resource = doc.Data()
		break
	}

	return resource, nil
}
