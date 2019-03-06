package v1

import v1 "github.com/olamai/simulation/pkg/api/v1"

// Add a spectator channel to the server
func (s *simulationServiceServer) AddSpectatorChannel(id string) string {
	// id := uuid.Must(uuid.NewV4()).String()
	s.spectIdChanMap[id] = make(chan v1.CellUpdate, 100)
	return id
}

// Remove a spectator channel from the server AND all it's subscriptions
func (s *simulationServiceServer) RemoveSpectatorChannel(id string) {
	// Loop over regions
	for region, spectatorIds := range s.spectRegionSubs {
		// If the user is subscribed to this region, remove their subscription
		for i, spectatorId := range spectatorIds {
			if spectatorId == id {
				s.spectRegionSubs[region] = append(spectatorIds[:i], spectatorIds[i+1:]...)
				break
			}
		}
	}
	delete(s.spectIdChanMap, id)
}

// Check if a spectator is already subbed to a region
func (s *simulationServiceServer) isSpectatorAlreadySubscribedToRegion(spectatorId string, region Vec2) bool {
	// Get subs for this region
	subs := s.spectRegionSubs[region]
	// Loop over and send to channel
	for _, _spectatorId := range subs {
		if _spectatorId == spectatorId {
			return true
		}
	}
	return false
}
