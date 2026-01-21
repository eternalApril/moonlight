package server

import (
	"fmt"
	"strings"

	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/store"
)

type Engine struct {
	commands map[string]Command
	storage  *store.Storage
}

func NewEngine(s store.Storage) *Engine {
	engine := Engine{
		commands: make(map[string]Command),
		storage:  &s,
	}
	engine.registerBasicCommand()

	return &engine
}

func (e *Engine) register(name string, cmd Command) {
	e.commands[strings.ToUpper(name)] = cmd
}

func (e *Engine) registerBasicCommand() {
	e.register("GET", CommandFunc(get))
	e.register("SET", CommandFunc(set))
	e.register("DEL", CommandFunc(del))
	e.register("PING", CommandFunc(ping))
	e.register("COMMAND", CommandFunc(command))
	e.register("TTL", CommandFunc(ttl))
	e.register("PTTL", CommandFunc(pttl))
}

func (e *Engine) Execute(name string, args []resp.Value) resp.Value {
	cmd, ok := e.commands[strings.ToUpper(name)]
	if !ok {
		return resp.MakeError(fmt.Sprintf("wrong command: %s", name))
	}

	ctx := &Context{
		args:    args,
		storage: e.storage,
	}

	return cmd.Execute(ctx)
}
