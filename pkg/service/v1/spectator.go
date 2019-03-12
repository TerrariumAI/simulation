package v1

import v1 "github.com/olamai/simulation/pkg/api/v1"

// Add a spectator channel to the server
func (s *simulationServiceServer) AddSpectatorChannel(id string) string {
	// id := uuid.Must(uuid.NewV4()).String()
	s.spectIDChanMap[id] = make(chan v1.CellUpdate, 100)
	return id
}

// Remove a spectator channel from the server AND all it's subscriptions
func (s *simulationServiceServer) RemoveSpectatorChannel(id string) {
	// Loop over regions
	for region, spectatorIDs := range s.spectRegionSubs {
		// If the user is subscribed to this region, remove their subscription
		for i, spectatorID := range spectatorIDs {
			if spectatorID == id {
				s.spectRegionSubs[region] = append(spectatorIDs[:i], spectatorIDs[i+1:]...)
				break
			}
		}
	}
	delete(s.spectIDChanMap, id)
}

// Check if a spectator is already subbed to a region
func (s *simulationServiceServer) isSpectatorAlreadySubscribedToRegion(spectatorID string, region Vec2) bool {
	// Get subs for this region
	subs := s.spectRegionSubs[region]
	// Loop over and send to channel
	for _, _spectatorID := range subs {
		if _spectatorID == spectatorID {
			return true
		}
	}
	return false
}
