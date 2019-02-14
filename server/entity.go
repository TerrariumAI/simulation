package main

import uuid "github.com/satori/go.uuid"

type Entity struct {
	Id     string
	Class  string
	Pos    Vec2
	Energy int32
	Health int32
}

const initial_energy = 100
const initial_health = 100

func NewEntity(class string, pos Vec2) Entity {
	return Entity{uuid.Must(uuid.NewV4()).String(), class, pos, initial_energy, initial_health}
}
