package main

import uuid "github.com/satori/go.uuid"

type Entity struct {
	id     string
	class  string
	pos    Vec2
	energy int32
	health int32
}

const initial_energy = 100
const initial_health = 100

func NewEntity(class string, pos Vec2) Entity {
	return Entity{uuid.Must(uuid.NewV4()).String(), class, pos, initial_energy, initial_health}
}
