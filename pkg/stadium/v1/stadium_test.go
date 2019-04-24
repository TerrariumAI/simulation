package stadium

import (
	"testing"

	v1 "github.com/olamai/simulation/pkg/api/v1"
	"github.com/olamai/simulation/pkg/vec2/v1"
	"github.com/olamai/simulation/pkg/world/v1"
)

const (
	regionSize = 16
)

func TestAddSpectator(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	s.AddSpectator(id)
	if _, ok := s.spectIDChanMap[id]; !ok {
		t.Error("Spectator does not exist after calling AddSpectator")
	}
}

func TestIsSpectatorSubscribedToRegion(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	region := vec2.Vec2{X: 0, Y: 0}
	// Test without spectator subbed
	isSubbed, _ := s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if isSubbed {
		t.Error("Spectator is subscribed to region, expected it not to be subscribed")
	}
	// Manually add this id to the region
	s.spectRegionSubs[region] = []string{id}
	isSubbed, _ = s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if !isSubbed {
		t.Error("Spectator is not subscribed to region, expected it to be subscribed")
	}
	// Remove the id from the region array
	s.spectRegionSubs[region] = []string{}
	isSubbed, _ = s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if isSubbed {
		t.Error("Spectator is subscribed to region, expected it not to be subscribed")
	}
}

func TestSubscribeSpectatorToRegion(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	region := vec2.Vec2{X: 0, Y: 0}
	s.SubscribeSpectatorToRegion(id, region)
	alreadySubbed, _ := s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if !alreadySubbed {
		t.Error("Spectator is not subscribed to region, expected it to be subscribed")
	}
	wasSubSuccessfull := s.SubscribeSpectatorToRegion(id, region)
	if wasSubSuccessfull {
		t.Error("Double sub was successfull, expected it to be unsuccessfull")
	}
}

func TestUnsubscribeSpectatorFromRegion(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	region := vec2.Vec2{X: 0, Y: 0}
	s.SubscribeSpectatorToRegion(id, region)
	isSubbed, _ := s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if !isSubbed {
		t.Error("Spectator is not subscribed to region, expected it to be subscribed")
	}
	s.UnsubscribeSpectatorFromRegion(id, region)
	isSubbed, _ = s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if isSubbed {
		t.Error("Spectator is subscribed to region, expected it to not be subscribed after unsub")
	}
}

func TestRemoveSpectator(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	region := vec2.Vec2{X: 0, Y: 0}
	// Add the new spectator
	s.AddSpectator(id)
	if _, ok := s.spectIDChanMap[id]; !ok {
		t.Error("Spectator does not exist after calling AddSpectator, expected it to exist")
	}
	// Subscribe to a region to make sure removing spectators removes them from regions
	s.SubscribeSpectatorToRegion(id, region)
	// Remove the spectator
	s.RemoveSpectator(id)
	if _, ok := s.spectIDChanMap[id]; ok {
		t.Error("Spectator still exists after calling RemoveSpectator, expected it to be removed")
	}
	// Make sure they were unsubbed
	isSubbed, _ := s.IsSpectatorAlreadySubscribedToRegion(id, region)
	if isSubbed {
		t.Error("Spectator is subscribed to region, expected it to not be subscribed after unsub")
	}
}

func TestBroadcastServerAction(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	expectedAction := "ACTION"
	// Add the new spectator
	channel := s.AddSpectator(id)
	s.BroadcastServerAction(expectedAction)
	serverActionResp := v1.SpectateResponse{}
	select {
	case serverActionResp = <-channel:
		break
	default:
		t.Error("Channel received nothing after broadcast, expected something")
	}
	serverAction := serverActionResp.Data.(*v1.SpectateResponse_ServerAction)
	action := serverAction.ServerAction.Action
	if action != expectedAction {
		t.Errorf("ServerActionResp data was %v, expected %v", action, expectedAction)
	}
}

func TestBroadcastCellUpdate(t *testing.T) {
	s := NewStadium(regionSize)
	id := "test-id"
	region := vec2.Vec2{X: 0, Y: 0}
	expectedPos := vec2.Vec2{X: 1, Y: 1}
	expectedEntityClass := "AGENT"
	var expectedEntityID int64
	expectedEntity := v1.Entity{Id: expectedEntityID, Class: expectedEntityClass}
	entity := world.Entity{Pos: expectedPos, Class: expectedEntityClass}
	// Add the new spectator
	channel := s.AddSpectator(id)
	s.SubscribeSpectatorToRegion(id, region)
	s.BroadcastCellUpdate(expectedPos, &entity)
	cellUpdateResp := v1.SpectateResponse{}
	select {
	case cellUpdateResp = <-channel:
		break
	default:
		t.Error("Channel received nothing after broadcast, expected something")
	}
	cellUpdate := cellUpdateResp.Data.(*v1.SpectateResponse_CellUpdate)
	cellUpdateEntity := cellUpdate.CellUpdate.Entity
	if cellUpdateEntity == nil {
		t.Error("Entity in cell update response was nil")
	}
	if cellUpdateEntity.Class != expectedEntity.Class {
		t.Errorf("CellUpdate entity class was %v, expected %v", cellUpdateEntity.Class, expectedEntity.Class)
	}
	if cellUpdateEntity.Id != expectedEntity.Id {
		t.Errorf("CellUpdate entity class was %v, expected %v", cellUpdateEntity.Id, expectedEntity.Id)
	}
}

func TestSendCellUpdate(t *testing.T) {
	s := NewStadium(regionSize)
	id1 := "test-id-1"
	id2 := "test-id-2"
	expectedPos := vec2.Vec2{X: 1, Y: 1}
	expectedEntityClass := "AGENT"
	var expectedEntityID int64
	expectedEntity := v1.Entity{Id: expectedEntityID, Class: expectedEntityClass}
	entity := world.Entity{Pos: expectedPos, Class: expectedEntityClass}
	// Add the new spectators
	channel1 := s.AddSpectator(id1)
	channel2 := s.AddSpectator(id2)
	s.SendCellUpdate(id1, expectedPos, &entity)
	// Get responses from both channels, only 1 should receive
	cellUpdateResp1 := v1.SpectateResponse{}
	// Make sure channel1 gets something
	select {
	case cellUpdateResp1 = <-channel1:
		break
	default:
		t.Error("Channel1 received nothing after broadcast, expected something")
	}
	// Make sure channel2 gets nothing
	select {
	case <-channel2:
		t.Error("Channel2 received something after broadcast, expected nothing")
	default:
		break
	}
	// Make sure channel1 received the correct data
	cellUpdate := cellUpdateResp1.Data.(*v1.SpectateResponse_CellUpdate)
	cellUpdateEntity := cellUpdate.CellUpdate.Entity
	if cellUpdateEntity == nil {
		t.Error("Entity in cell update response was nil")
	}
	if cellUpdateEntity.Class != expectedEntity.Class {
		t.Errorf("CellUpdate entity class was %v, expected %v", cellUpdateEntity.Class, expectedEntity.Class)
	}
	if cellUpdateEntity.Id != expectedEntity.Id {
		t.Errorf("CellUpdate entity class was %v, expected %v", cellUpdateEntity.Id, expectedEntity.Id)
	}
}
