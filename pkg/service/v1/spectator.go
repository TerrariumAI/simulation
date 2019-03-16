package v1

import v1 "github.com/olamai/simulation/pkg/api/v1"

// Add a spectator channel to the server
func (s *simulationServiceServer) addSpectatorChannel(id string) string {
	// id := uuid.Must(uuid.NewV4()).String()
	s.spectIDChanMap[id] = make(chan v1.SpectateResponse, 100)
	return id
}

// Remove a spectator channel from the server AND all it's subscriptions
func (s *simulationServiceServer) removeSpectatorChannel(id string) {
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
func (s *simulationServiceServer) isSpectatorAlreadySubscribedToRegion(spectatorID string, region Vec2) (isAlreadySubbed bool, index int) {
	// Get subs for this region
	subs := s.spectRegionSubs[region]
	// Loop over and send to channel
	for i, _spectatorID := range subs {
		if _spectatorID == spectatorID {
			return true, i
		}
	}
	return false, -1
}

func (s *simulationServiceServer) broadcastServerAction(action string) {
	for _, channel := range s.spectIDChanMap {
		channel <- v1.SpectateResponse{
			Data: &v1.SpectateResponse_ServerAction{
				&v1.ServerAction{
					Action: action,
				},
			},
		}
	}
}

// Broadcast a cell update
func (s *simulationServiceServer) broadcastCellUpdate(pos Vec2, entity *Entity) {
	// Get region for this position
	region := pos.getRegion()
	// Get subs for this region
	subs := s.spectRegionSubs[region]
	// Loop over and send to channel
	for _, spectatorID := range subs {
		channel := s.spectIDChanMap[spectatorID]
		if entity == nil {
			channel <- v1.SpectateResponse{
				Data: &v1.SpectateResponse_CellUpdate{
					&v1.CellUpdate{
						X:      pos.x,
						Y:      pos.y,
						Entity: nil,
					},
				},
			}
		} else {
			channel <- v1.SpectateResponse{
				Data: &v1.SpectateResponse_CellUpdate{
					&v1.CellUpdate{
						X: pos.x,
						Y: pos.y,
						Entity: &v1.Entity{
							Id:    entity.id,
							Class: entity.class,
						},
					},
				},
			}
		}
	}
}
