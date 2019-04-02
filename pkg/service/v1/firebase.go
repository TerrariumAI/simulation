package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/olamai/simulation/pkg/logger"

	"go.uber.org/zap"

	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/metadata"
)

const mockSecret = "MOCK-SECRET"

// Initialize a new firebase app instance
func initializeFirebaseApp(env string) *firebase.App {
	serviceAccountFileLocation := "./serviceAccountKey.json"
	// -----------------------------------
	// ENV CHECK
	// -----------------------------------
	//Return a testing token with fake uid
	if env == "training" {
		return nil
	}
	if env == "testing" {
		serviceAccountFileLocation = "./serviceAccountKey_testing.json"
	}
	// -----------------------------------
	// Initialize firebase app
	opt := option.WithCredentialsFile(serviceAccountFileLocation)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		logger.Log.Fatal("error initializing firebase app: %v\n", zap.String("reason", err.Error()))
		return nil
	}
	return app
}

// func authenticateFirebaseAccountWithIDToken(ctx context.Context, app *firebase.App, env string) *auth.Token {
// 	// get the auth token from the context
// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		return nil
// 	}
// 	authTokenHeader, ok := md["auth-token"]
// 	if !ok {
// 		logger.Log.Warn("verifyFirebaseIDToken(): No auth-token header in context")
// 		return nil
// 	}
// 	idToken := authTokenHeader[0]
// 	// -----------------------------------
// 	// TESTING FUNCTIONALITY
// 	// -----------------------------------
// 	if env != "prod" {
// 		// If this is the correct testing token, return a testing token with fake uid
// 		if idToken == "TEST-ID-TOKEN" {
// 			return &auth.Token{
// 				UID: "TEST-UID",
// 			}
// 		}
// 		// If not correct test token, return nil
// 		return nil
// 	}
// 	// -----------------------------------
// 	// Make sure the firebase app instance exists
// 	if app == nil {
// 		logger.Log.Warn("Couldn't authenticate user: error initializing firebase app")
// 		return nil
// 	}
// 	// Attempt to create a firebase auth client
// 	client, err := app.Auth(context.Background())
// 	if err != nil {
// 		logger.Log.Warn("Error getting Auth client: %v\n", zap.String("reason", err.Error()))
// 		return nil
// 	}
// 	// Verify the token
// 	token, err := client.VerifyIDToken(ctx, idToken)
// 	if err != nil {
// 		logger.Log.Warn("Error verifying ID token: %v\n", zap.String("reason", err.Error()))
// 		return nil
// 	}

// 	return token
// }

func authenticateFirebaseAccountWithSecret(ctx context.Context, app *firebase.App, env string) (map[string]interface{}, error) {
	// get the auth token from the context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}
	secretHeader, ok := md["auth-secret"]
	if !ok {
		logger.Log.Warn("authenticateFirebaseAccountWithSecret(): No secret token header in context")
		return nil, errors.New("Missing Secret Key In Metadata")
	}
	secret := secretHeader[0]

	// -----------------------------------
	// ENVIRONMENT CHECK
	// -----------------------------------
	// Training,  doesn't implement authentication
	if env == "training" {
		if secret == mockSecret {
			fakeUser := make(map[string]interface{})
			fakeUser["id"] = "FAKE_USER_ID"
			return fakeUser, nil
		}
		return nil, errors.New("Invalid Secret Key")
	}
	// -----------------------------------

	// Create a firestore client
	client, err := app.Firestore(context.Background())
	defer client.Close()
	if err != nil {
		return nil, err
	}
	// Query for the user
	iter := client.Collection("users").Where("secret", "==", secret).Documents(context.Background())
	dsnap, err := iter.Next()
	if err == iterator.Done {
		return nil, errors.New("Invalid Secret Key")
	}
	if err != nil {
		return nil, err
	}
	// Add the UID to the user data
	m := dsnap.Data()
	m["id"] = dsnap.Ref.ID
	return m, nil
}

func addRemoteModelToFirebase(app *firebase.App, uid string, name string, env string) error {
	if env == "training" {
		return nil
	}
	// Create the client
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

func removeRemoteModelFromFirebase(app *firebase.App, uid string, name string, env string) error {
	if env == "training" {
		return nil
	}
	// Create the client
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

func removeAllRemoteModelsFromFirebase(app *firebase.App, env string) error {
	if env == "training" {
		return nil
	}
	// Create the client
	client, err := app.Firestore(context.Background())
	defer client.Close()
	if err != nil {
		logger.Log.Warn("removeAllRemoteModelsFromFirebase(): Error creating Firestore client")
		fmt.Println(err)
		return err
	}
	// Make sure we can add the new RM
	iter := client.Collection("remoteModels").Documents(context.Background())
	snaps, err := iter.GetAll()
	if err != nil {
		logger.Log.Warn("removeAllRemoteModelsFromFirebase(): Error getting all remoteModels")
		fmt.Println(err)
		return err
	}
	for _, snap := range snaps {
		snap.Ref.Delete(context.Background())
	}
	return nil
}
