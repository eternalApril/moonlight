package server

import (
	"github.com/eternalApril/moonlight/internal/resp"
)

func command(ctx *Context) resp.Value {
	// must return docs in the future
	return resp.MakeSimpleString("OK")
}

func ping(ctx *Context) resp.Value {
	// command takes zero or one arguments
	if len(ctx.args) > 1 {
		return resp.MakeErrorWrongNumberOfArguments("PING")
	}

	if len(ctx.args) == 1 {
		return resp.MakeBulkString(string(ctx.args[0].String))
	}

	return resp.MakeSimpleString("PONG")
}

func get(ctx *Context) resp.Value {
	return resp.MakeSimpleString("GET")
}

func set(ctx *Context) resp.Value {
	return resp.MakeSimpleString("SET")
}
