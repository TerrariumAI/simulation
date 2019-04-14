package world

import (
	"testing"

	"github.com/olamai/simulation/pkg/vec2/v1"
)

func TestNewFoodEntity(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	expectedPos := vec2.Vec2{X: 0, Y: 0}
	expectedClass := "FOOD"

	agent, err := w.NewFoodEntity(expectedPos)

	if err != nil {
		t.Errorf("Got error creating new agent in empty world: %v", err)
	}
	if agent.Class != expectedClass {
		t.Errorf("New entity class was %v, expected %v", expectedClass, agent.Class)
	}
	if agent.Pos != expectedPos {
		t.Errorf("New entity position was %v, expected %v", expectedPos, agent.Pos)
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestNewAgentEntity(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	expectedPos := vec2.Vec2{X: 0, Y: 0}
	expectedClass := "AGENT"

	agent, err := w.NewAgentEntity("", "", expectedPos)

	if err != nil {
		t.Errorf("Got error creating new agent in empty world: %v", err)
	}
	if agent.Class != expectedClass {
		t.Errorf("New agent class was %v, expected AGENT", agent.Class)
	}
	if agent.Pos != expectedPos {
		t.Errorf("New agent position was %v, expected 0,0", agent.Pos)
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestNewEntityOnOccupiedCell(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	_, err := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})

	if err == nil {
		t.Error("Expected error when creating 2 agents in same position but got none")
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestDeleteEntity(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 2
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	deleted := w.DeleteEntity(agent.ID)

	if deleted == false {
		t.Error("DeleteEntity returned false, it was not deleted")
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestEntityMove(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 3
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	moved := w.EntityMove(agent.ID, vec2.Vec2{X: 0, Y: 1})

	expectedPosition := vec2.Vec2{X: 0, Y: 1}

	if moved == false {
		t.Error("EntityMove returned false, expected true")
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
	if agent.Pos != expectedPosition {
		t.Errorf("Entity position is %v, expected %v", agent.Pos, expectedPosition)
	}
}

func TestEntityConsume(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 3
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	agent.Energy = 80
	w.NewFoodEntity(vec2.Vec2{X: 1, Y: 0})
	consumed := w.EntityConsume(agent.ID, vec2.Vec2{X: 1, Y: 0})

	expectedEnergy := 90

	if consumed == false {
		t.Error("EntityConsume returned false, expected true")
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
	if agent.Energy != int32(expectedEnergy) {
		t.Errorf("Entity energy is %v, expected %v", agent.Energy, expectedEnergy)
	}
}

func TestAgentLivingCostUpdate(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})

	// Test living cost at full energy
	w.EntityLivingCostUpdate(agent)
	expectedEnergy := initialEnergy - entityLivingEnergyCost
	expectedHealth := 100
	if agent.Energy != int32(expectedEnergy) {
		t.Errorf("Energy is at %v, expected %v", agent.Energy, expectedEnergy)
	}
	if agent.Health != int32(expectedHealth) {
		t.Errorf("Energy is at %v, expected %v", agent.Health, expectedHealth)
	}

	// Test living cost at 0 energy
	agent.Energy = 0
	w.EntityLivingCostUpdate(agent)
	expectedEnergy = 0
	expectedHealth = initialHealth - entityNoEnergyHealthCost
	if agent.Energy != int32(expectedEnergy) {
		t.Errorf("Energy is at %v, expected %v", agent.Energy, expectedEnergy)
	}
	if agent.Health != int32(expectedHealth) {
		t.Errorf("Energy is at %v, expected %v", agent.Health, expectedHealth)
	}

	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestDoesEntityExist(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	exists := w.DoesEntityExist(agent.ID)

	if exists == false {
		t.Error("DoesEntityExist returned false, expected true")
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestGetEntity(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	entity := w.GetEntity(agent.ID)

	if entity == nil {
		t.Errorf("Got entity %v, expected %v", entity, agent)
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestGetEntityByPos(t *testing.T) {
	cellUpdateCount := 0
	expectedCellUpdateCount := 1
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	entity := w.GetEntityByPos(agent.Pos)

	if entity == nil {
		t.Errorf("Got entity %v, expected %v", entity, agent)
	}
	if entity.ID != agent.ID {
		t.Errorf("Got entity with id %v, expected %v", entity.ID, agent.ID)
	}
	if cellUpdateCount != expectedCellUpdateCount {
		t.Errorf("Cell updated count was %v, expected %v", cellUpdateCount, expectedCellUpdateCount)
	}
}

func TestReset(t *testing.T) {
	cellUpdateCount := 0
	w := NewWorld(
		16,
		func(vec2.Vec2, *Entity) {
			cellUpdateCount++
		},
		false)

	agent, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 0, Y: 0})
	agent2, _ := w.NewAgentEntity("", "", vec2.Vec2{X: 1, Y: 0})
	w.Reset()
	agent = w.GetEntity(agent.ID)
	agent2 = w.GetEntity(agent2.ID)

	if agent != nil {
		t.Errorf("Got entity %v, expected %v", agent, nil)
	}
	if agent2 != nil {
		t.Errorf("Got entity %v, expected %v", agent2, nil)
	}
}
