package server

import (
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/store"
)

type Context struct {
	args    []resp.Value
	storage *store.Storage
}

type Command interface {
	Execute(ctx *Context) resp.Value
}

type CommandFunc func(ctx *Context) resp.Value

func (c CommandFunc) Execute(ctx *Context) resp.Value {
	return c(ctx)
}
