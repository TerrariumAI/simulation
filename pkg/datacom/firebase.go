package datacom

import (
	"context"
	"errors"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
)

// GetRemoteModelMetadataBySecret checks the database to see if a remote model exists,
// if so returns metadata
func (dc *Datacom) GetRemoteModelMetadataBySecret(modelSecret string) (*RemoteModel, error) {
	// Training case
	if dc.env == "training" {
		return &RemoteModel{
			ID:           "MOCK-MODEL-ID",
			OwnerUID:     "MOCK-UID",
			ConnectCount: 1,
		}, nil
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
		err := errors.New("remote model does not exist")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	var remoteModel RemoteModel
	docs[0].DataTo(&remoteModel)
	remoteModel.ID = docs[0].Ref.ID

	return &remoteModel, nil
}

// GetRemoteModelMetadataByID checks the database to see if a remote model exists,
// if so returns metadata
func (dc *Datacom) GetRemoteModelMetadataByID(modelID string) (*RemoteModel, error) {
	// Training case
	if dc.env == "training" {
		return &RemoteModel{
			ID:           "MOCK-MODEL-ID",
			OwnerUID:     "MOCK-UID",
			ConnectCount: 1,
		}, nil
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
	remoteModel.ID = dsnap.Ref.ID

	return &remoteModel, nil
}

// UpdateRemoteModelMetadata updates a remote model's metadata
func (dc *Datacom) UpdateRemoteModelMetadata(remoteModelMD *RemoteModel, connectCount int) error {
	// Training case
	if dc.env == "training" {
		return nil
	}

	// Init client
	ctx := context.Background()
	client, err := dc.firebaseApp.Firestore(ctx)
	defer client.Close()
	if err != nil {
		return err
	}

	_, err = client.Collection("remoteModels").Doc(remoteModelMD.ID).Set(ctx, map[string]interface{}{
		"connectCount": connectCount,
	}, firestore.MergeAll)

	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred updating model id=%s: %s", remoteModelMD.ID, err)
		return err
	}

	return nil
}
