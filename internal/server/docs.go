package server

import (
	"strings"

	"github.com/eternalApril/moonlight/internal/resp"
)

type commandMetadata struct {
	arity    int      // Arity includes the command name itself
	flags    []string // read, write, fast, denyoom, etc
	firstKey int      // 1-based index of the first key
	lastKey  int      // 1-based index of the last key
	step     int      // Step count for finding keys
}

var (
	commandRegistry = map[string]commandMetadata{
		"PING":    {-1, []string{"fast", "stale"}, 0, 0, 0},
		"GET":     {2, []string{"readonly", "fast"}, 1, 1, 1},
		"SET":     {-3, []string{"write", "denyoom"}, 1, 1, 1},
		"DEL":     {-2, []string{"write"}, 1, -1, 1},
		"TTL":     {2, []string{"readonly", "fast"}, 1, 1, 1},
		"PTTL":    {2, []string{"readonly", "fast"}, 1, 1, 1},
		"PERSIST": {2, []string{"write", "fast"}, 1, 1, 1},
		"COMMAND": {-1, []string{"random", "loading", "stale"}, 0, 0, 0},
	}
)

// commandDoc stores a description for the command
type commandDoc struct {
	summary    string
	complexity string
	group      string
	since      string
}

// commandDocsRegistry documentation registry
var commandDocsRegistry = map[string]commandDoc{
	"PING": {
		summary:    "Ping the server.",
		complexity: "O(1)",
		group:      "connection",
		since:      "1.0.0",
	},
	"GET": {
		summary:    "Get the value of a key.",
		complexity: "O(1)",
		group:      "string",
		since:      "1.0.0",
	},
	"SET": {
		summary:    "Set the string value of a key.",
		complexity: "O(1)",
		group:      "string",
		since:      "1.0.0",
	},
	"DEL": {
		summary:    "Delete a key.",
		complexity: "O(N) where N is the number of keys that will be removed.",
		group:      "generic",
		since:      "1.0.0",
	},
	"TTL": {
		summary:    "Get the time to live for a key in seconds.",
		complexity: "O(1)",
		group:      "generic",
		since:      "1.0.0",
	},
	"PTTL": {
		summary:    "Get the time to live for a key in milliseconds.",
		complexity: "O(1)",
		group:      "generic",
		since:      "1.0.0",
	},
	"PERSIST": {
		summary:    "Remove the expiration from a key.",
		complexity: "O(1)",
		group:      "generic",
		since:      "1.0.0",
	},
	"COMMAND": {
		summary:    "Get array of command details.",
		complexity: "O(N) where N is the number of commands to look up.",
		group:      "server",
		since:      "1.0.0",
	},
}

func makeFlagsArray(flags []string) resp.Value {
	vals := make([]resp.Value, len(flags))
	for i, f := range flags {
		vals[i] = resp.MakeSimpleString(f)
	}
	return resp.MakeArray(vals)
}

func makeInfoCmdArray(name string) []resp.Value {
	return []resp.Value{
		resp.MakeBulkString(name),
		resp.MakeInteger(int64(commandRegistry[name].arity)),
		makeFlagsArray(commandRegistry[name].flags),
		resp.MakeInteger(int64(commandRegistry[name].firstKey)),
		resp.MakeInteger(int64(commandRegistry[name].lastKey)),
		resp.MakeInteger(int64(commandRegistry[name].step)),
	}
}

func getAllCommands() resp.Value {
	cmdArray := make([]resp.Value, 0, len(commandRegistry))
	for name := range commandRegistry {
		details := makeInfoCmdArray(name)
		cmdArray = append(cmdArray, resp.MakeArray(details))
	}
	return resp.MakeArray(cmdArray)
}

// getCommandsDocs returns documentation for specified commands or all commands
// Format: [Name, [Summary, val, Since, val...], Name, [...]]
func getCommandsDocs(args []resp.Value) resp.Value {
	var targets []string

	if len(args) == 0 {
		targets = make([]string, 0, len(commandDocsRegistry))
		for name := range commandDocsRegistry {
			targets = append(targets, name)
		}
	} else {
		targets = make([]string, 0, len(args))
		for _, arg := range args {
			targets = append(targets, strings.ToUpper(string(arg.String)))
		}
	}

	result := make([]resp.Value, 0, len(targets)*2)

	for _, name := range targets {
		doc, ok := commandDocsRegistry[name]
		if !ok {
			continue
		}

		result = append(result, resp.MakeBulkString(name))

		props := []resp.Value{
			resp.MakeBulkString("summary"),
			resp.MakeBulkString(doc.summary),
			resp.MakeBulkString("since"),
			resp.MakeBulkString(doc.since),
			resp.MakeBulkString("group"),
			resp.MakeBulkString(doc.group),
			resp.MakeBulkString("complexity"),
			resp.MakeBulkString(doc.complexity),
		}

		result = append(result, resp.MakeArray(props))
	}

	return resp.MakeArray(result)
}
