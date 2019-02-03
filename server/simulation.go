package main

import (
	"time"
)

func (s *Server) startSimulation() {
	// Repeat every frame
	for {

		// Check if there is anything in the channel
		if len(s.chAgentSpawn) > 0 {
			// Handle every element that is currently in the channel
			for i := 0; i < len(s.chAgentSpawn); i++ {
				// Get the SpawnAgentWithResultChan object from channel
				spawnAgentWithResultChan := <-s.chAgentSpawn
				// Parse out the message that was sent
				spawnAgentMsg := spawnAgentWithResultChan.msg
				chNewAgentId := spawnAgentWithResultChan.chNewAgentId
				// Create the new entity
				id := s.GetEntityId()
				s.agents[id] = &Entity{Id: id, Class: "agent", Pos: vec2{X: spawnAgentMsg.X, Y: spawnAgentMsg.Y}}
				// Send the new id to the channel
				chNewAgentId <- id
			}
		}

		// TODO - make this only run in dev mode
		// Go through agent actions and perform each one
		if len(s.chAgentAction) > 0 {
			for i := 0; i < len(s.chAgentAction); i++ {
				// Pop message
				agentSpawnMsg := <-s.chAgentAction
				id := agentSpawnMsg.Id
				action := agentSpawnMsg.Action
				println("Action: ", action, "ID: ", id)
				// Get agent
				agent := s.agents[id]
				// Perform action
				switch action {
				case "UP":
					agent.Pos.Y += 1
				case "DOWN":
					agent.Pos.Y -= 1
				case "RIGHT":
					agent.Pos.X += 1
				case "LEFT":
					agent.Pos.X -= 1
				}
			}
		}

		// Perform Agent actions
		for _, agent := range s.agents {
			println("Sending to clients: ", agent.Pos.X, agent.Pos.Y)
			for _, obsvChan := range s.observerationChannels {
				obsvChan <- EntityUpdate{Action: "update", Entity: *agent}
			}
		}

		// Sleep
		time.Sleep(50 * time.Millisecond)
	}
}
