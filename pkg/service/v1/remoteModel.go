package v1

import (
	"time"

	v1 "github.com/olamai/simulation/pkg/api/v1"
)

const fps = 5

// Add a remote model channel to the server
func (s *simulationServiceServer) addRemoteModelChannel(uid string) chan v1.Observation {
	// id := uuid.Must(uuid.NewV4()).String()
	channel := make(chan v1.Observation, 100)
	s.remoteModelMap[uid] = channel
	return channel
}

// Remove a remote model channel from the server
func (s *simulationServiceServer) removeRemoteModelChannel(uid string) {
	delete(s.remoteModelMap, uid)
}

func (s *simulationServiceServer) canUserAddRemoteModel(uid string) bool {
	if _, ok := s.remoteModelMap[uid]; ok {
		return false
	}
	return true
}

func (s *simulationServiceServer) remoteModelStepper() {
	for {
		// Lock the data, defer unlock until end of call
		s.m.Lock()
		for _, agent := range s.agents {
			ownerUID := agent.ownerUID
			remoteModelChan, ok := s.remoteModelMap[ownerUID]
			// If this agent agent's RM doesn't exist yet/anymore, continue
			if !ok {
				continue
			}
			// Get and send the observation to the RM
			cells := s.getObservationCellsForPosition(agent.pos)
			remoteModelChan <- v1.Observation{
				Id:     agent.id,
				Alive:  true,
				Cells:  cells,
				Energy: agent.energy,
				Health: agent.health,
			}
		}
		// Unlock the data
		s.m.Unlock()
		// Sleep
		time.Sleep((1000 / fps) * time.Millisecond)
	}

}
