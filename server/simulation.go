package main

import (
	"time"
)

func (s *Server) startSimulation() {
	// Repeat every frame
	for {

		// Check if there is anything in the channel
		if len(s.chSpawnAgent) > 0 {
			// Handle every element that is currently in the channel
			for i := 0; i < len(s.chSpawnAgent); i++ {
				// Get the SpawnAgentWithResultChan object from channel
				spawnAgentWithResultChan := <-s.chSpawnAgent
				// Parse out the message that was sent
				spawnAgentMsg := spawnAgentWithResultChan.msg
				chNewAgentId := spawnAgentWithResultChan.chNewAgentId
				// Create the new entity
				id := s.GetEntityId()
				s.agents = append(s.agents, &Entity{Id: id, Class: "agent", Pos: vec2{X: spawnAgentMsg.X, Y: spawnAgentMsg.Y}})
				// Send the new id to the channel
				chNewAgentId <- id
			}
		}

		// Perform Agent actions
		for _, agent := range s.agents {
			// TODO - change this to send to a channel
			agent.Pos.X += 1
			println("Sending to clients: ", agent.Pos.X, agent.Pos.Y)
			for _, obsvChan := range s.observerationChannels {
				obsvChan <- EntityUpdate{Action: "update", Entity: *agent}
			}
		}

		// Sleep
		time.Sleep(50 * time.Millisecond)
	}
}
