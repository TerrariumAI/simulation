package v1

import (
	"context"
	"errors"

	"github.com/olamai/simulation/pkg/logger"

	"go.uber.org/zap"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/metadata"
)

// Initialize a new firebase app instance
func initializeFirebaseApp(env string) *firebase.App {
	// -----------------------------------
	// TESTING FUNCTIONALITY
	// -----------------------------------
	//Return a testing token with fake uid
	if env != "prod" {
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
	// -----------------------------------
	// TESTING FUNCTIONALITY
	// -----------------------------------
	if env != "prod" {
		// If this is the correct testing token, return a testing token with fake uid
		if idToken == "TEST-ID-TOKEN" {
			return &auth.Token{
				UID: "TEST-UID",
			}
		}
		// If not correct test token, return nil
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

func getUserProfileWithSecret(ctx context.Context, app *firebase.App) (map[string]interface{}, error) {
	// get the auth token from the context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}
	authTokenHeader, ok := md["auth-token"]
	if !ok {
		logger.Log.Warn("getUserProfileWithSecret(): No auth-token header in context")
		return nil, nil
	}
	secret := authTokenHeader[0]

	sa := option.WithCredentialsFile("./serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, sa)
	if err != nil {
		return nil, err
	}
	client, err := app.Firestore(context.Background())
	defer client.Close()
	if err != nil {
		return nil, err
	}
	iter := client.Collection("users").Where("secret", "==", secret).Documents(context.Background())
	dsnap, err := iter.Next()
	if err == iterator.Done {
		return nil, errors.New("Invalid Secret Key")
	}
	if err != nil {
		return nil, err
	}
	m := dsnap.Data()
	m["id"] = dsnap.Ref.ID
	return m, nil
}

func addRemoteModelToFirebase(app *firebase.App, uid string, name string) error {
	// Create the client
	sa := option.WithCredentialsFile("./serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, sa)
	if err != nil {
		return err
	}
	client, err := app.Firestore(context.Background())
	defer client.Close()
	if err != nil {
		return err
	}
	// Make sure we can add the new RM
	iter := client.Collection("remoteModels").Where("user", "==", uid).Where("name", "==", name).Documents(context.Background())
	snaps, err := iter.GetAll()
	if err != nil {
		return err
	}
	if len(snaps) != 0 {
		return errors.New("Remote model with that name already exists")
	}
	// Add the RM
	_, _, err = client.Collection("remoteModels").Add(context.Background(), map[string]interface{}{
		"name": name,
		"user": uid,
	})
	if err != nil {
		return err
	}
	return nil
}

func removeRemoteModelFromFirebase(app *firebase.App, uid string, name string) error {
	// Create the client
	sa := option.WithCredentialsFile("./serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, sa)
	if err != nil {
		return err
	}
	client, err := app.Firestore(context.Background())
	defer client.Close()
	if err != nil {
		return err
	}
	// Make sure we can add the new RM
	iter := client.Collection("remoteModels").Where("user", "==", uid).Where("name", "==", name).Documents(context.Background())
	snaps, err := iter.GetAll()
	for _, snap := range snaps {
		snap.Ref.Delete(context.Background())
	}
	return nil
}

func removeAllRemoteModelsFromFirebase(app *firebase.App) error {
	// Create the client
	sa := option.WithCredentialsFile("./serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, sa)
	if err != nil {
		return err
	}
	client, err := app.Firestore(context.Background())
	defer client.Close()
	if err != nil {
		return err
	}
	// Make sure we can add the new RM
	iter := client.Collection("remoteModels").Documents(context.Background())
	snaps, err := iter.GetAll()
	for _, snap := range snaps {
		snap.Ref.Delete(context.Background())
	}
	return nil
}
