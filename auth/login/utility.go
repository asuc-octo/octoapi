package login

import (
	"context"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var jwtKeyResourceID = "projects/980046983693/secrets/jwt_encryption_key/versions/1"

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
