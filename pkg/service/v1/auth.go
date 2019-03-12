package v1

import (
	"context"
	"fmt"

	"github.com/olamai/simulation/pkg/logger"

	"go.uber.org/zap"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"google.golang.org/api/option"
	"google.golang.org/grpc/metadata"
)

// Initialize a new firebase app instance
func initializeFirebaseApp(env string) *firebase.App {
	// -----------------------------------
	// TESTING FUNCTIONALITY
	// -----------------------------------
	//Return a testing token with fake uid
	if env == "testing" || env == "debug" {
		return nil
	}
	// -----------------------------------
	// Initialize firebase app
	opt := option.WithCredentialsFile("./serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		logger.Log.Fatal("error initializing firebase app: %v\n", zap.String("reason", err.Error()))
		return nil
	}
	return app
}

func verifyFirebaseIDToken(ctx context.Context, app *firebase.App, env string) *auth.Token {
	// get the auth token from the context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	authTokenHeader, ok := md["auth-token"]
	if !ok {
		logger.Log.Warn("verifyFirebaseIDToken(): No auth-token header in context")
		return nil
	}
	idToken := authTokenHeader[0]
	fmt.Println("ID TOKEN FOUND: " + idToken)
	// -----------------------------------
	// TESTING FUNCTIONALITY
	// -----------------------------------
	if env == "testing" || env == "debug" {
		// If this is the correct testing token, return a testing token with fake uid
		if idToken == "TEST-ID-TOKEN" {
			return &auth.Token{
				UID: "TEST-UID",
			}
		}
		// If not, return nil
		return nil
	}
	// -----------------------------------
	// Make sure the firebase app instance exists
	if app == nil {
		logger.Log.Warn("Couldn't authenticate user: error initializing firebase app")
		return nil
	}
	// Attempt to create a firebase auth client
	client, err := app.Auth(context.Background())
	if err != nil {
		logger.Log.Warn("Error getting Auth client: %v\n", zap.String("reason", err.Error()))
		return nil
	}
	// Verify the token
	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		logger.Log.Warn("Error verifying ID token: %v\n", zap.String("reason", err.Error()))
		return nil
	}

	return token
}
