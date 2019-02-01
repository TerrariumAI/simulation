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
				spawnAgentMsg := <-s.chSpawnAgent
				s.agents = append(s.agents, &Entity{Class: "agent", Pos: vec2{X: spawnAgentMsg.X, Y: spawnAgentMsg.Y}})
			}
		}

		// Perform Agent actions
		for _, agent := range s.agents {
			agent.Pos.X += 1
			println(agent.Pos.X, agent.Pos.Y)
		}

		// Sleep
		time.Sleep(50 * time.Millisecond)
	}
}
