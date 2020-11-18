package dining

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// type Dining struct {
// 	Name string `json:"name"`
//     Latitude float64 `json:"latitude"`
//     Longitude float64 `json:"longitude"`
//     Phone string `json:"phone"`
//     Description string `json:"description"`
//     Address string `json:"address"`
// }

var DiningFields = [6]string{"name", "description", "latitude", "longitude", "address", "phone"}
var client *firestore.Client
var ctx context.Context
var firestoreKeyResourceID = "projects/980046983693/secrets/firestore_access_key/versions/1"

func DiningEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fstoreErr := initFirestore(w)
	if fstoreErr != nil {
		http.Error(w, "Couldn’t connect to database", http.StatusInternalServerError)
		log.Printf("Firestore Init failed: %v", fstoreErr)
		return
	}
	dinings, diningErr := listDinings(ctx, w, client)
	if diningErr != nil {
		http.Error(w, "Couldn’t connect to database", http.StatusInternalServerError)
		log.Printf("dining GET failed: %v", diningErr)
		return
	}
	output, jsonErr := json.Marshal(&dinings)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
		log.Printf("libraries JSON conversion failed: %v", jsonErr)
		return
	}
	fmt.Fprint(w, string(output))
}

func initFirestore(w http.ResponseWriter) error {
	ctx = context.Background()
	/* Get Auth for accessing Firestore by getting firestore secret */
	key, err := getFirestoreSecret(w)
	if err != nil {
		return err
	}
	/* Load Firestore */
	var clientErr error
	opt := option.WithCredentialsJSON([]byte(key))
	client, clientErr = firestore.NewClient(ctx, "berkeley-mobile", opt)
	if clientErr != nil {
		return clientErr
	}
	return nil
}

func listDinings(ctx context.Context, w http.ResponseWriter, client *firestore.Client) ([]map[string]interface{}, error) {
	defer client.Close()
	var dinings []map[string]interface{}
	iter := client.Collection("Dining Halls").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docData := doc.Data()
		dining := make(map[string]interface{})
		for _, element := range DiningFields {
			dining[element] = docData[element]
		}
		dinings = append(dinings, dining)
	}
	return dinings, nil
}

func getFirestoreSecret(w http.ResponseWriter) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: firestoreKeyResourceID,
	}
	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", err
	}
	return string(result.Payload.Data), nil
}
