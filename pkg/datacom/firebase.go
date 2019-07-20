package datacom

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// GetRemoteModelMetadataBySecret checks the database to see if a remote model exists,
// if so returns metadata
func (dc *Datacom) GetRemoteModelMetadataBySecret(modelSecret string) (*RemoteModel, error) {
	// Test case
	if dc.env == "testing" {
		if modelSecret == "MOCK-SECRET" {
			return &RemoteModel{
				ID:           "MOCK-MODEL-ID",
				ConnectCount: 1,
			}, nil
		}
		return nil, errors.New("That RM does does not exist")
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

	return &remoteModel, nil
}

// GetRemoteModelMetadataByID checks the database to see if a remote model exists,
// if so returns metadata
func (dc *Datacom) GetRemoteModelMetadataByID(modelID string) (*RemoteModel, error) {
	// Test case
	if dc.env == "testing" {
		if modelID == "MOCK-MODEL-ID" {
			return &RemoteModel{
				ID:           "MOCK-MODEL-ID",
				OwnerID:      "MOCK-UID",
				ConnectCount: 1,
			}, nil
		}
		return nil, errors.New("That RM does does not exist")
	}

	// Init client
	ctx := context.Background()
	client, err := dc.firebaseApp.Firestore(ctx)
	defer client.Close()
	if err != nil {
		return nil, err
	}

	// Try to get the RM
	dsnap, err := client.Collection("remoteModels").Doc(modelID).Get(ctx)
	if err != nil {
		err := fmt.Errorf("Invalid model id: %v", err)
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	var remoteModel RemoteModel
	dsnap.DataTo(&remoteModel)

	return &remoteModel, nil
}

// UpdateRemoteModelMetadata updates a remote model's metadata
func (dc *Datacom) UpdateRemoteModelMetadata(id string, connectCount int32) {
	// TODO:
	log.Println("WARNING: UpdateRemoteModelMetadata() UNIMPLEMENTED METHOD")
}
