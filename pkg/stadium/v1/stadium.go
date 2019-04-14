package stadium

import (
	v1 "github.com/olamai/simulation/pkg/api/v1"
	"github.com/olamai/simulation/pkg/vec2/v1"
	"github.com/olamai/simulation/pkg/world/v1"
)

// Stadium handles spectators and their region subscriptions
type Stadium struct {
	// Map from spectator id -> observation channel
	spectIDChanMap map[string]chan v1.SpectateResponse
	// Specators subscription to regions
	spectRegionSubs map[vec2.Vec2][]string
}

// NewStadium creates a new stadium
func NewStadium() Stadium {
	return Stadium{
		spectIDChanMap:  make(map[string]chan v1.SpectateResponse),
		spectRegionSubs: make(map[vec2.Vec2][]string),
	}
}

// AddSpectator adds a spectator channel to the server
func (s *Stadium) AddSpectator(id string) chan v1.SpectateResponse {
	channel := make(chan v1.SpectateResponse, 100)
	s.spectIDChanMap[id] = channel
	return channel
}

// DoesSpectatorExist Checks if a spectator exists by this id
func (s *Stadium) DoesSpectatorExist(id string) bool {
	_, ok := s.spectIDChanMap[id]
	return ok
}

// IsSpectatorAlreadySubscribedToRegion checks if a spectator is already subbed to a region
func (s *Stadium) IsSpectatorAlreadySubscribedToRegion(spectatorID string, region vec2.Vec2) (isAlreadySubbed bool, index int) {
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

// SubscribeSpectatorToRegion adds the spectator to a region
func (s *Stadium) SubscribeSpectatorToRegion(id string, region vec2.Vec2) bool {
	alreadySubbed, _ := s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if alreadySubbed {
		return false
	}
	s.spectRegionSubs[region] = append(s.spectRegionSubs[region], id)
	return true
}

// UnsubscribeSpectatorFromRegion removes the spectator from a region
func (s *Stadium) UnsubscribeSpectatorFromRegion(id string, region vec2.Vec2) bool {
	alreadySubbed, i := s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if !alreadySubbed {
		return false
	}
	s.spectRegionSubs[region] = append(s.spectRegionSubs[region][:i], s.spectRegionSubs[region][i+1:]...)
	// Remove the region key if there are no more spectators in the region
	if len(s.spectRegionSubs[region]) == 0 {
		delete(s.spectRegionSubs, region)
	}
	return true
}

// RemoveSpectator removes a spectator channel from the server AND all it's subscriptions
func (s *Stadium) RemoveSpectator(id string) {
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

// BroadcastServerAction broadcasts a Server Action to everyone on this server. This is usually
//  a WORLDRESET message
func (s *Stadium) BroadcastServerAction(action string) {
	for _, channel := range s.spectIDChanMap {
		channel <- v1.SpectateResponse{
			Data: &v1.SpectateResponse_ServerAction{
				ServerAction: &v1.ServerAction{
					Action: action,
				},
			},
		}
	}
}

// BroadcastCellUpdate broadcasts a cell update only to those listening on that specific region
func (s *Stadium) BroadcastCellUpdate(pos vec2.Vec2, regionSize int32, entity *world.Entity) {
	// Get region for this position
	region := pos.GetRegion(regionSize)
	// Get subs for this region
	subs := s.spectRegionSubs[region]
	// Loop over and send to channel
	for _, spectatorID := range subs {
		channel := s.spectIDChanMap[spectatorID]
		if entity == nil {
			channel <- v1.SpectateResponse{
				Data: &v1.SpectateResponse_CellUpdate{
					CellUpdate: &v1.CellUpdate{
						X:      pos.X,
						Y:      pos.Y,
						Entity: nil,
					},
				},
			}
		} else {
			channel <- v1.SpectateResponse{
				Data: &v1.SpectateResponse_CellUpdate{
					CellUpdate: &v1.CellUpdate{
						X: pos.X,
						Y: pos.Y,
						Entity: &v1.Entity{
							Id:    entity.ID,
							Class: entity.Class,
						},
					},
				},
			}
		}
	}
}

// SendCellUpdate sends a cell update to a specific spectator
func (s *Stadium) SendCellUpdate(id string, pos vec2.Vec2, entity *world.Entity) bool {
	channel, ok := s.spectIDChanMap[id]
	if !ok {
		return false
	}
	if entity == nil {
		channel <- v1.SpectateResponse{
			Data: &v1.SpectateResponse_CellUpdate{
				CellUpdate: &v1.CellUpdate{
					X:      pos.X,
					Y:      pos.Y,
					Entity: nil,
				},
			},
		}
	} else {
		channel <- v1.SpectateResponse{
			Data: &v1.SpectateResponse_CellUpdate{
				CellUpdate: &v1.CellUpdate{
					X: pos.X,
					Y: pos.Y,
					Entity: &v1.Entity{
						Id:    entity.ID,
						Class: entity.Class,
					},
				},
			},
		}
	}
	return true
}
