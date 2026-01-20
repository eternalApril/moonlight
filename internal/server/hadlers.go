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
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("GET")
	}

	value, ok := (*ctx.storage).Get(string(ctx.args[0].String))
	if !ok {
		return resp.MakeNilBulkString()
	}

	return resp.MakeBulkString(value)
}

func set(ctx *Context) resp.Value {
	// implement the processing of all flags and TTL
	if len(ctx.args) != 2 {
		return resp.MakeErrorWrongNumberOfArguments("SET")
	}

	key := ctx.args[0].String
	value := ctx.args[1].String

	(*ctx.storage).Set(string(key), string(value))

	return resp.MakeSimpleString("OK")
}

func del(ctx *Context) resp.Value {
	if len(ctx.args) < 1 {
		return resp.MakeErrorWrongNumberOfArguments("DEL")
	}

	wasDeleted := 0
	for _, key := range ctx.args {
		if (*ctx.storage).Delete(string(key.String)) {
			wasDeleted++
		}
	}

	return resp.MakeInteger(wasDeleted)
}
