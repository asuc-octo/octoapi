package login

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	// "github.com/auth0/go-jwt-middleware"
	firebase "firebase.google.com/go"
	"github.com/dgrijalva/jwt-go"

	"cloud.google.com/go/firestore"

	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
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
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,PATCH,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Headers", "*")

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

	defaultApp, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		http.Error(w, "Error while initializing the app.", http.StatusBadRequest)
		return
	}
	defaultClient, err := defaultApp.Auth(context.Background())
	if err != nil {
		http.Error(w, "Error while setting the default app", http.StatusBadRequest)
		return
	}

	user, err := defaultClient.GetUser(ctx, uid)
	if err != nil {
		http.Error(w, "Error while verifying the id", http.StatusBadRequest)
		return
	}

	email := user.UserInfo.Email
	if !strings.Contains(email, "berkeley.edu") {
		http.Error(w, "Error while verifying user has berkeley email.", http.StatusBadRequest)
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
	sendEmail(email, string(tokensJSON))
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

func sendEmail(email string, tokens string) error {
	emailSecret, emailSecretGenErr := getEmailSecret()
	if emailSecretGenErr != nil {
		return emailSecretGenErr
	}
	from := mail.NewEmail("OCTO API", "octo.api@asuc.org")
	subject := "tokens for OCTO API Authorization"
	to := mail.NewEmail("", email)
	plainTextContent := tokens
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, plainTextContent)
	emailclient := sendgrid.NewSendClient(emailSecret)
	_, emailErr := emailclient.Send(message)
	if emailErr != nil {
		return emailErr
	}
	return nil
}


0x8048446