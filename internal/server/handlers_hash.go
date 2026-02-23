package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/storage"
)

// hset sets the specified fields to their respective values in the hash stored at key
func hset(ctx *context) resp.Value {
	if len(ctx.args) < 3 || len(ctx.args)%2 != 1 {
		return resp.MakeErrorWrongNumberOfArguments("HSET")
	}

	fields := make(map[string]string, len(ctx.args)/2)

	for i := 1; i != len(ctx.args); i += 2 {
		fields[string(ctx.args[i].String)] = string(ctx.args[i+1].String)
	}

	created := (*ctx.storage).HSet(string(ctx.args[0].String), fields)

	return resp.MakeInteger(created)
}

// hget returns the value associated with field in the hash stored at key
func hget(ctx *context) resp.Value {
	if len(ctx.args) != 2 {
		return resp.MakeErrorWrongNumberOfArguments("HGET")
	}

	str, ok := (*ctx.storage).HGet(string(ctx.args[0].String), string(ctx.args[1].String))
	if !ok {
		return resp.MakeNilBulkString()
	}
	return resp.MakeBulkString(str)
}

// hgetall returns all fields and values of the hash stored at key
func hgetall(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("HGETALL")
	}

	mp := (*ctx.storage).HGetAll(string(ctx.args[0].String))
	return resp.MakeMap(mp)
}

// hdel parse arguments for storage.HDel
func hdel(ctx *context) resp.Value {
	if len(ctx.args) < 2 {
		return resp.MakeErrorWrongNumberOfArguments("HDEL")
	}

	key := string(ctx.args[0].String)
	fields := make([]string, len(ctx.args)-1)

	for i, field := range ctx.args[1:] {
		fields[i] = string(field.String)
	}

	deleted := (*ctx.storage).HDel(key, fields)

	return resp.MakeInteger(deleted)
}

// hexists parse arguments for storage.HExists
func hexists(ctx *context) resp.Value {
	if len(ctx.args) != 2 {
		return resp.MakeErrorWrongNumberOfArguments("HEXISTS")
	}

	key := string(ctx.args[0].String)
	field := string(ctx.args[1].String)

	exist := (*ctx.storage).HExists(key, field)

	return resp.MakeInteger(exist)
}

// hlen parse arguments for storage.HLen
func hlen(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("HLEN")
	}

	key := string(ctx.args[0].String)

	mapLen := (*ctx.storage).HLen(key)

	return resp.MakeInteger(mapLen)
}

// hkeys parse arguments for storage.HKeys
func hkeys(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("HKEYS")
	}

	key := string(ctx.args[0].String)

	fields := (*ctx.storage).HKeys(key)
	response := make([]resp.Value, 0, len(fields))
	for _, field := range fields {
		response = append(response, resp.MakeSimpleString(field))
	}

	return resp.MakeArray(response)
}

// hvals parse arguments for storage.HVals
func hvals(ctx *context) resp.Value {
	if len(ctx.args) != 1 {
		return resp.MakeErrorWrongNumberOfArguments("HVALS")
	}

	key := string(ctx.args[0].String)

	vals := (*ctx.storage).HVals(key)
	response := make([]resp.Value, 0, len(vals))
	for _, val := range vals {
		response = append(response, resp.MakeSimpleString(val))
	}

	return resp.MakeArray(response)
}

// hexpire HEXPIRE key seconds [NX|XX|GT|LT] FIELDS numfields field [field ...]
func hexpire(ctx *context) resp.Value {
	if len(ctx.args) < 4 {
		return resp.MakeErrorWrongNumberOfArguments("HEXPIRE")
	}

	key := string(ctx.args[0].String)

	secStr := string(ctx.args[1].String)
	seconds, err := strconv.ParseInt(secStr, 10, 64)
	if err != nil {
		return resp.MakeError("value is not an integer or out of range")
	}
	ttl := time.Duration(seconds) * time.Second

	opts := storage.ExpireOptions{}
	fieldsIdx := -1

	for i := 2; i < len(ctx.args); i++ {
		arg := strings.ToUpper(string(ctx.args[i].String))
		if arg == "FIELDS" {
			fieldsIdx = i
			break
		}

		switch arg {
		case "NX":
			opts.NX = true
		case "XX":
			opts.XX = true
		case "GT":
			opts.GT = true
		case "LT":
			opts.LT = true
		default:
			return resp.MakeError(fmt.Sprintf("Unsupported option %s", arg))
		}
	}

	if (opts.NX && opts.XX) || (opts.GT && opts.LT) || (opts.NX && (opts.GT || opts.LT)) {
		return resp.MakeError("ERR NX and XX, GT or LT options at the same time are not compatible")
	}

	if fieldsIdx == -1 {
		return resp.MakeError("ERR syntax error, missing FIELDS")
	}

	if fieldsIdx+1 >= len(ctx.args) {
		return resp.MakeError("ERR numfields missing")
	}

	numFieldsStr := string(ctx.args[fieldsIdx+1].String)
	numFields, err := strconv.Atoi(numFieldsStr)
	if err != nil {
		return resp.MakeError("value is not an integer or out of range")
	}

	if fieldsIdx+2+numFields > len(ctx.args) {
		return resp.MakeError("parameter count mismatch")
	}

	fields := make([]string, 0, numFields)
	for i := 0; i < numFields; i++ {
		fields = append(fields, string(ctx.args[fieldsIdx+2+i].String))
	}

	resCodes, ok := (*ctx.storage).HExpire(key, ttl, opts, fields)

	if !ok {
		resCodes = make([]int, len(fields))
		for i := range resCodes {
			resCodes[i] = -2
		}
	}

	respArr := make([]resp.Value, len(resCodes))
	for i, c := range resCodes {
		respArr[i] = resp.MakeInteger(int64(c))
	}

	return resp.MakeArray(respArr)
}
