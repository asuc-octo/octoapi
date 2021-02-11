package gyms

import (
	"context"
	"net/http"

	"github.com/dgrijalva/jwt-go"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var firestoreKeyResourceID = "projects/980046983693/secrets/firestore_access_key/versions/1"
var jwtKeyResourceID = "projects/980046983693/secrets/jwt_encryption_key/versions/1"

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

func getJwtSecret() ([]byte, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: jwtKeyResourceID,
	}
	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	return []byte(string(result.Payload.Data)), nil
}

func validateAccessToken(r *http.Request) bool {
	accessHeader := r.Header.Get("Authorization")
	if len(accessHeader) < 6 {
		return false
	}
	accesstoken := accessHeader[7:]
	claims := jwt.MapClaims{}
	jwtTokenSecret, err := getJwtSecret()
	if err != nil {
		return false
	}
	_, parsingerr := jwt.ParseWithClaims(accesstoken, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtTokenSecret, nil
	})
	if parsingerr != nil {
		return false
	}
	if claims["type"] != "access" {
		return false
	}
	return true
}
