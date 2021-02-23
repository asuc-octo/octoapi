package refreshtoken

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	// "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"

	"cloud.google.com/go/firestore"
)

type User struct {
	Uid          string `json:"uid"`
	Refreshtoken string `json:"refresh-token"`
	Created_at   int    `json:"created_at"`
	Blocked      bool   `json:"blocked"`
	blocked_at   int    `json:"created_at"`
}

type Tokens struct {
	AccessToken string `json:"access-token"`
}

func RefreshAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Something went wrong. Please make sure you are passing your refresh token in the request body as {“refresh-token”: ‘<token>’}.", http.StatusBadRequest)
		return
	}
	var data map[string]interface{}
	jsonErr := json.Unmarshal([]byte(reqBody), &data)
	if jsonErr != nil {
		http.Error(w, "Something went wrong. Please make sure you are passing your refresh token in the request body as {“refresh-token”: ‘<token>’}.", http.StatusBadRequest)
		return
	}
	refreshToken, converttoken := data["refresh-token"].(string)
	if !converttoken {
		http.Error(w, "Something went wrong. Please make sure you are passing your refresh token in the request body as {“refresh-token”: ‘<token>’}.", http.StatusBadRequest)
		return
	}
	uid, decodeErr := decodeRefreshToken(refreshToken)
	if decodeErr != nil {
		http.Error(w, "Something went wrong. Please make sure you are passing your refresh token in the request body as {“refresh-token”: ‘<token>’}.", http.StatusBadRequest)
		return
	}

	client, ctx, clientErr := initFirestore(w)
	if clientErr != nil {
		http.Error(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		log.Printf("firestore init failed: %v", clientErr)
		return
	}
	token, tokenErr := getAccessToken(uid, client, ctx)
	if tokenErr != nil {
		http.Error(w, tokenErr.Error(), http.StatusBadRequest)
		return
	}
	tokens := Tokens{token}
	tokensJSON, jsonErr := json.Marshal(tokens)
	if jsonErr != nil {
		http.Error(w, "Something went wrong. Please make sure you are passing your refresh token in the request body as {“refresh-token”: ‘<token>’}.", http.StatusBadRequest)
		log.Printf("token generation failed: %v", jsonErr)
		return
	}
	w.Write(tokensJSON)
}

func decodeRefreshToken(refreshtoken string) (string, error) {
	claims := jwt.MapClaims{}

	jwtTokenSecret, jwterr := getJwtSecret()
	if jwterr != nil {
		return "", jwterr
	}

	_, err := jwt.ParseWithClaims(refreshtoken, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtTokenSecret, nil
	})
	if err != nil {
		return "", err
	}
	if claims["type"] != "refresh" {
		return "", errors.New("invalid refresh token")
	}
	return claims["uid"].(string), nil
}

func getAccessToken(uid string, client *firestore.Client, ctx context.Context) (string, error) {
	defer client.Close()

	// first check if user already exists in the database
	userQuery, queryErr := client.Collection("users").Doc(uid).Get(ctx)
	if queryErr != nil {
		return "", errors.New("Something went wrong. Please make sure you are passing your refresh token in the request body as {“refresh-token”: ‘<token>’}.")
	}
	userData := userQuery.Data()
	if userData["blocked"].(bool) {
		return "", errors.New("Your account has been blocked. If you believe something went wrong, please contact octo.api@asuc.org for details.")
	}
	newJwtToken, tokenGenErr := getAccessToken(uid)
	if tokenGenErr != nil {
		return "", tokenGenErr
	}
	return newJwtToken, nil
}

func getAccessToken(uid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":  uid,
		"type": "access",
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})
	jwtTokenSecret, err := getJwtSecret()
	if err != nil {
		return "", err
	}
	tokenString, err := token.SignedString(jwtTokenSecret)
	return tokenString, err
}

func initFirestore(w http.ResponseWriter) (*firestore.Client, context.Context, error) {
	// Use the application default credentials
	ctx := context.Background()
	client, clientErr := firestore.NewClient(ctx, "api-team-292919")
	if clientErr != nil {
		return nil, nil, clientErr
	}
	return client, ctx, nil
}
