package v1

import (
	"errors"

	v1 "github.com/terrariumai/simulation/pkg/api/v1"
)

const fps = 5

type remoteModel struct {
	name    string
	channel chan v1.Observation
}

// Add a remote model channel to the server memory and DB
func (s *simulationServiceServer) addRemoteModel(uid string, name string) (*remoteModel, error) {
	// Add remote model to firestore
	err := addRemoteModelToFirebase(s.firebaseApp, uid, name, s.env)
	if err != nil {
		return nil, errors.New("CreateRemoteModel(): Model with that name already exists")
	}
	// Create the new channel and RM
	channel := make(chan v1.Observation, 100)
	newRM := &remoteModel{
		name,
		channel,
	}
	// Add the RM
	s.remoteModelMap[uid] = append(s.remoteModelMap[uid], newRM)
	return newRM, nil
}

// Remove a remote model channel from the server memory and DB
func (s *simulationServiceServer) removeRemoteModel(uid string, name string) bool {
	userRMs := s.remoteModelMap[uid]
	// Find the RM and remove it
	for i, RM := range userRMs {
		if RM.name == name {
			// Remove the remote model from the server
			s.remoteModelMap[uid] = append(userRMs[:i], userRMs[i+1:]...)
			// Remove the remote model from the DB
			removeRemoteModelFromFirebase(s.firebaseApp, uid, name, s.env)
			return true
		}
	}
	return false
}

// Checks if a remote model exists ONLY in server memory
func (s *simulationServiceServer) doesRemoteModelExist(uid string, name string) bool {
	userRMs := s.remoteModelMap[uid]
	// Find the RM and remove it
	for _, RM := range userRMs {
		if RM.name == name {
			return true
		}
	}
	return false
}
