package dining

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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

func DiningSearchEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := strings.ToLower(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "Url Param 'name' is missing", http.StatusBadRequest)
		return
	}
	client, ctx, fstoreErr := initFirestore(w)
	if fstoreErr != nil {
		http.Error(w, fstoreErr.Error(), http.StatusInternalServerError)
		return
	}
	dinings, diningErr := searchDinings(ctx, w, client, name)
	if diningErr != nil {
		http.Error(w, diningErr.Error(), http.StatusInternalServerError)
		return
	}
	output, jsonErr := json.Marshal(&dinings)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(output))
}

func initFirestore(w http.ResponseWriter) (*firestore.Client, context.Context, error) {
	ctx := context.Background()

	/* Get Auth for accessing Firestore by getting json file in cloud storage*/
	storageClient, clientErr := storage.NewClient(ctx)
	if clientErr != nil {
		return nil, nil, clientErr
	}
	defer storageClient.Close()
	bkt := storageClient.Bucket("firestore_access")
	obj := bkt.Object("berkeley-mobile-e0922919475f.json")
	read, readErr := obj.NewReader(ctx)
	if readErr != nil {
		return nil, nil, readErr
	}
	defer read.Close()
	json_input := StreamToByte(read) // the byte array of the json file

	/* Load Firestore */
	opt := option.WithCredentialsJSON(json_input)
	client, new_err := firestore.NewClient(ctx, "berkeley-mobile", opt)
	if new_err != nil {
		return nil, nil, new_err
	}
	return client, ctx, nil
}

func searchDinings(ctx context.Context, w http.ResponseWriter, client *firestore.Client, name string) ([]map[string]interface{}, error) {
	defer client.Close()
	dinings := make([]map[string]interface{}, 0)
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
		if strings.Contains(strings.ToLower(fmt.Sprintf("%s", docData["name"])), name) {
			dining := make(map[string]interface{})
			for _, element := range DiningFields {
				dining[element] = docData[element]
			}
			dinings = append(dinings, dining)
		}
	}
	return dinings, nil
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
