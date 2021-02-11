package login

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
	RefreshToken string `json:"refresh-token"`
	AccessToken  string `json:"access-token"`
}

func AuthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid body params", http.StatusBadRequest)
		return
	}
	var data map[string]interface{}
	jsonErr := json.Unmarshal([]byte(reqBody), &data)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusBadRequest)
		return
	}
	uid, convertuid := data["uid"].(string)
	if !convertuid {
		http.Error(w, "Invalid uid params", http.StatusBadRequest)
		return
	}
	client, ctx, clientErr := initFirestore(w)
	if clientErr != nil {
		http.Error(w, clientErr.Error(), http.StatusInternalServerError)
		log.Printf("firestore init failed: %v", clientErr)
		return
	}
	refreshToken, accessToken, tokenErr := getTokens(uid, client, ctx)
	if tokenErr != nil {
		http.Error(w, tokenErr.Error(), http.StatusBadRequest)
		log.Printf("token generation failed: %v", tokenErr)
		return
	}
	tokens := Tokens{refreshToken, accessToken}
	tokensJSON, jsonErr := json.Marshal(tokens)
	if jsonErr != nil {
		http.Error(w, "Couldn’t get tokens", http.StatusInternalServerError)
		log.Printf("token generation failed: %v", jsonErr)
		return
	}
	w.Write(tokensJSON)
}

func getTokens(uid string, client *firestore.Client, ctx context.Context) (string, string, error) {
	defer client.Close()

	// first check if user already exists in the database
	userQuery, queryErr := client.Collection("users").Doc(uid).Get(ctx)
	if queryErr == nil {
		var user User
		if getDataErr := userQuery.DataTo(&user); getDataErr != nil {
			return "", "", getDataErr
		}
		if user.Blocked {
			return "", "", errors.New("user is blocked")
		}
		accessJwtToken, accessTokenGenErr := getAccessJwtToken(uid)
		if accessTokenGenErr != nil {
			return "", "", accessTokenGenErr
		}
		return user.Refreshtoken, accessJwtToken, nil
	}
	newJwtToken, tokenGenErr := getRefreshJwtToken(uid)
	if tokenGenErr != nil {
		return "", "", tokenGenErr
	}
	_, addErr := client.Collection("users").Doc(uid).Set(ctx, map[string]interface{}{
		"uid":          uid,
		"created_at":   time.Now().Unix(),
		"refreshtoken": newJwtToken,
		"blocked":      false,
		"blocked_at":   nil,
	})
	if addErr != nil {
		return "", "", addErr
	}
	accessJwtToken, accessTokenGenErr := getAccessJwtToken(uid)
	if accessTokenGenErr != nil {
		return "", "", accessTokenGenErr
	}

	return newJwtToken, accessJwtToken, nil
}

func getRefreshJwtToken(uid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":  uid,
		"type": "refresh",
	})
	jwtTokenSecret, err := getJwtSecret()
	if err != nil {
		return "", err
	}
	tokenString, err := token.SignedString(jwtTokenSecret)
	return tokenString, err
}

func getAccessJwtToken(uid string) (string, error) {
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
