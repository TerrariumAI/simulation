package main

import uuid "github.com/satori/go.uuid"

type Entity struct {
	Id     string
	Class  string
	Pos    Vec2
	Energy int32
}

const entity_initial_energy = 100

func NewEntity(class string, pos Vec2) Entity {
	return Entity{uuid.Must(uuid.NewV4()).String(), class, pos, entity_initial_energy}
}
