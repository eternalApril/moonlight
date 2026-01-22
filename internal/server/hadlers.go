package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	if len(ctx.args) < 2 {
		return resp.MakeErrorWrongNumberOfArguments("SET")
	}

	key := string(ctx.args[0].String)
	value := string(ctx.args[1].String)

	var (
		nx, xx bool
		TTL    time.Duration
		hasTTL bool
	)

	for i := 2; i != len(ctx.args); i++ {
		arg := strings.ToUpper(string(ctx.args[i].String))

		switch arg {
		case "NX":
			if xx {
				return resp.MakeError("NX cannot use with XX")
			}

			nx = true
		case "XX":
			if nx {
				return resp.MakeError("XX cannot use with NX")
			}

			xx = true
		case "EX", "PX", "EXAT", "PXAT":
			if hasTTL {
				return resp.MakeError("cannot specify the TTL twice")
			}

			if i+1 >= len(ctx.args) {
				return resp.MakeError("syntax error")
			}

			valTTLStr := ctx.args[i+1].String
			valTTL, err := strconv.ParseInt(string(valTTLStr), 10, 64)
			if err != nil {
				return resp.MakeError("value TTL is not integer or out of range")
			}

			switch arg {
			case "EX":
				TTL = time.Duration(valTTL) * time.Second
			case "PX":
				TTL = time.Duration(valTTL) * time.Millisecond
			case "EXAT":
				expireAt := time.Unix(valTTL, 0)
				TTL = time.Until(expireAt)
			case "PXAT":
				expireAt := time.UnixMilli(valTTL)
				TTL = time.Until(expireAt)
			}

			fmt.Printf("TTL: %v\n", TTL)

			if TTL <= 0 && (arg == "EXAT" || arg == "PXAT") {
				(*ctx.storage).Set(key, value, -1)
				return resp.MakeSimpleString("OK")
			}

			hasTTL = true
			i++
		default:
			return resp.MakeError(fmt.Sprintf("syntax error with command: %s", arg))
		}
	}

	// TODO flags realize
	(*ctx.storage).Set(key, value, 0)

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

func ttl(ctx *Context) resp.Value {
	return resp.Value{}
}

func pttl(ctx *Context) resp.Value {
	return resp.Value{}
}
