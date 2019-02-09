package main

import uuid "github.com/satori/go.uuid"

type Entity struct {
	Id    string
	Class string
	Pos   Vec2
}

func NewEntity(class string, pos Vec2) Entity {
	return Entity{uuid.Must(uuid.NewV4()).String(), class, pos}
}
