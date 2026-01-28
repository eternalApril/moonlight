package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/storage"
)

func cmd(_ *context) resp.Value {
	// must return docs in the future
	return resp.MakeSimpleString("OK")
}

// ping returns PONG if no arguments are provided, or a copy of the argument if one is given
func ping(ctx *context) resp.Value {
	// command takes zero or one arguments
	if len(ctx.args) > 1 {
		return resp.MakeErrorWrongNumberOfArguments("PING")
	}

	if len(ctx.args) == 1 {
		return resp.MakeBulkString(string(ctx.args[0].String))
	}

	return resp.MakeSimpleString("PONG")
}

// get retrieves the value of a key. Returns a Nil Bulk String if the key does not exist
func get(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("GET")
	}

	value, ok := (*ctx.storage).Get(string(ctx.args[0].String))
	if !ok {
		return resp.MakeNilBulkString()
	}

	return resp.MakeBulkString(value)
}

// set assigns a value to a key with optional parameters
func set(ctx *context) resp.Value {
	if len(ctx.args) < 2 {
		return resp.MakeErrorWrongNumberOfArguments("SET")
	}

	key := string(ctx.args[0].String)
	value := string(ctx.args[1].String)

	options := storage.SetOptions{}

	// flag tracking to prevent syntax errors
	var hasTTL bool

	for i := 2; i != len(ctx.args); i++ {
		arg := strings.ToUpper(string(ctx.args[i].String))

		switch arg {
		case "NX":
			if options.XX {
				return resp.MakeError("NX cannot use with XX")
			}
			options.NX = true

		case "XX":
			if options.NX {
				return resp.MakeError("XX cannot use with NX")
			}
			options.XX = true

		case "KEEPTTL":
			if hasTTL {
				return resp.MakeError("TTL already specified")
			}
			options.KeepTTL = true
			hasTTL = true

		case "EX", "PX", "EXAT", "PXAT":
			if hasTTL {
				return resp.MakeError("TTL already specified")
			}

			if i+1 >= len(ctx.args) {
				return resp.MakeError("syntax error")
			}

			valTTLStr := ctx.args[i+1].String
			valTTL, err := strconv.ParseInt(string(valTTLStr), 10, 64)
			if err != nil {
				return resp.MakeError("value TTL is not integer or out of range")
			}

			var ttlDuration time.Duration

			switch arg {
			case "EX":
				ttlDuration = time.Duration(valTTL) * time.Second
			case "PX":
				ttlDuration = time.Duration(valTTL) * time.Millisecond
			case "EXAT":
				expireAt := time.Unix(valTTL, 0)
				ttlDuration = time.Until(expireAt)
			case "PXAT":
				expireAt := time.UnixMilli(valTTL)
				ttlDuration = time.Until(expireAt)
			}

			if ttlDuration <= 0 && (arg == "EXAT" || arg == "PXAT") {
				options.TTL = time.Duration(1) * time.Nanosecond
				(*ctx.storage).Set(key, value, options)
				return resp.MakeSimpleString("OK")
			}

			options.TTL = ttlDuration
			hasTTL = true
			i++
		default:
			return resp.MakeError(fmt.Sprintf("syntax error with command: %s", arg))
		}
	}

	ok := (*ctx.storage).Set(key, value, options)

	if !ok {
		return resp.MakeNilBulkString()
	}

	return resp.MakeSimpleString("OK")
}

// del removes the specified keys. Returns the number of keys that were removed
func del(ctx *context) resp.Value {
	if len(ctx.args) < 1 {
		return resp.MakeErrorWrongNumberOfArguments("DEL")
	}

	var wasDeleted int64 = 0
	for _, key := range ctx.args {
		if (*ctx.storage).Delete(string(key.String)) {
			wasDeleted++
		}
	}

	return resp.MakeInteger(wasDeleted)
}

// ttl returns the remaining time to live of a key in seconds
func ttl(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("TTL")
	}

	key := string(ctx.args[0].String)
	duration, code := (*ctx.storage).Expiry(key)

	if code < 0 {
		return resp.MakeInteger(int64(code))
	}

	return resp.MakeInteger(int64(duration.Seconds()))
}

// pttl returns the remaining time to live of a key in milliseconds
func pttl(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("PTTL")
	}

	key := string(ctx.args[0].String)
	duration, code := (*ctx.storage).Expiry(key)

	if code < 0 {
		return resp.MakeInteger(int64(code))
	}

	return resp.MakeInteger(duration.Milliseconds())
}

// persist removes the expiration from a key, making it persistent
func persist(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("PERSIST")
	}

	key := string(ctx.args[0].String)

	code := (*ctx.storage).Persist(key)

	return resp.MakeInteger(code)
}
