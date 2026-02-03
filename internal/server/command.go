package server

import (
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/storage"
)

// context every command gets this struct as an argument
type context struct {
	args    []resp.Value
	storage *storage.Storage
	peer    *Peer
}

// command defines a common interface for all executable server commands
type command interface {
	execute(ctx *context) resp.Value
}

// commandFunc is an adapter that allows you to use regular functions as commands
type commandFunc func(ctx *context) resp.Value

// execute implements the command interface for the commandFunc type
func (c commandFunc) execute(ctx *context) resp.Value {
	return c(ctx)
}
