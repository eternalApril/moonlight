package server

import (
	"strings"

	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/store"
)

type Engine struct {
	commands map[string]Command
	storage  *store.Storage
}

func NewEngine(s store.Storage) *Engine {
	return &Engine{
		commands: make(map[string]Command),
		storage:  &s,
	}
}

func (e *Engine) Register(name string, cmd Command) {
	e.commands[strings.ToUpper(name)] = cmd
}

func (e *Engine) Execute(name string, args []resp.Value) resp.Value {
	cmd, ok := e.commands[strings.ToUpper(name)]
	if !ok {
		return resp.Value{} //resp.MakeError
	}

	ctx := &Context{
		args:    args,
		storage: e.storage,
	}

	return cmd.Execute(ctx)
}
