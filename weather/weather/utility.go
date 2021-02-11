package weather

import (
	"context"
	"net/http"

	"github.com/dgrijalva/jwt-go"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var weatherKeyResourceID = "projects/980046983693/secrets/weather_access_key/versions/1"
var jwtKeyResourceID = "projects/980046983693/secrets/jwt_encryption_key/versions/1"

func getWeatherSecret(w http.ResponseWriter) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: weatherKeyResourceID,
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
