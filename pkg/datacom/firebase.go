package datacom

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// GetRemoteModelMetadataForUser checks the database to see if a remote model exists,
// if so returns metadata
func (dc *Datacom) GetRemoteModelMetadataForUser(modelSecret string) (*RemoteModel, error) {
	// Test case
	if dc.env == "testing" {
		if modelSecret == "MOCK-SECRET" {
			return &RemoteModel{
				ID: "MOCK-MODEL-ID",
			}, nil
		}
		return nil, errors.New("That RM does not belong to you or does not exist")
	}

	// Init client
	ctx := context.Background()
	client, err := dc.firebaseApp.Firestore(ctx)
	defer client.Close()
	if err != nil {
		return nil, err
	}

	// Try to get the RM
	q := client.Collection("remoteModels").Where("secretKey", "==", modelSecret).Limit(1)

	docs, err := q.Documents(ctx).GetAll()
	if err != nil {
		log.Println("error iter")
		return nil, fmt.Errorf("invalid secret key: %v", err)
	}
	if len(docs) == 0 {
		log.Println("zero results")
		return nil, fmt.Errorf("invalid secret key: %v", err)
	}

	var remoteModel RemoteModel
	docs[0].DataTo(&remoteModel)
	remoteModel.ID = modelSecret

	return &remoteModel, nil
}
