package server

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eternalApril/moonlight/internal/config"
	"github.com/eternalApril/moonlight/internal/persistence"
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/storage"
	"go.uber.org/zap"
)

// Engine coordinates the execution of commands and manages the background tasks of the repository
type Engine struct {
	commands map[string]command // Registry of available commands (the key is the command name in uppercase)
	storage  *storage.Storage   // Interface to the underlying KV storage
	cfg      *config.Config     // Configuration engine
	stopGC   chan struct{}      // Channel for the background GC stop signal
	stopOnce sync.Once          // Ensures that the stop happens only once
	aof      *persistence.AOF   // AOF instance
	rdb      *persistence.RDB   // RDB instance
	logger   *zap.Logger
}

// NewEngine initializes the engine, registers the basic commands, and
// if enabled in the config, starts background cleanup of outdated keys
func NewEngine(s storage.Storage, cfg *config.Config, logger *zap.Logger) (*Engine, error) {
	engine := Engine{
		commands: make(map[string]command),
		storage:  &s,
		cfg:      cfg,
		stopGC:   make(chan struct{}),
		logger:   logger,
	}
	engine.registerBasicCommand()

	if cfg.Persistence.AOF.Enabled {
		aof, err := persistence.NewAOF(
			cfg.Persistence.AOF.Filename,
			cfg.Persistence.AOF.Fsync,
			logger,
		)
		if err != nil {
			return nil, err
		}
		engine.aof = aof

		// Restore existing AOF
		engine.restoreAOF()
	}

	if cfg.Persistence.RDB.Enabled {
		engine.rdb = persistence.NewRDB(cfg.Persistence.RDB.Filename, logger)

		if !cfg.Persistence.AOF.Enabled {
			if err := engine.rdb.Load(s); err != nil {
				logger.Error("Failed to load RDB", zap.Error(err))
			}
		}

		if cfg.Persistence.RDB.Interval != "" {
			go engine.startAutoSave(cfg.Persistence.RDB.Interval)
		}
	}

	if cfg.GC.Enabled {
		go engine.startGCLoop()
	}

	return &engine, nil
}

func (e *Engine) startAutoSave(intervalStr string) {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		e.logger.Error("Invalid RDB interval", zap.Error(err))
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go func() {
				if err := e.rdb.Save(*e.storage); err != nil {
					e.logger.Error("Auto-save RDB failed", zap.Error(err))
				}
			}()
		case <-e.stopGC:
			return
		}
	}
}

func (e *Engine) restoreAOF() {
	cmds, err := e.aof.Load()
	if err != nil {
		e.logger.Error("Failed to load AOF", zap.Error(err))
		return
	}

	e.logger.Info("Restoring AOF...", zap.Int("commands", len(cmds)))

	for _, cmdVal := range cmds {
		if cmdVal.Type != resp.TypeArray || len(cmdVal.Array) == 0 {
			continue
		}

		name := string(cmdVal.Array[0].String)
		args := cmdVal.Array[1:]

		cmd, ok := e.commands[strings.ToUpper(name)]
		if ok {
			ctx := &context{args: args, storage: e.storage}
			cmd.execute(ctx)
		}
	}
	e.logger.Info("AOF restore finished")
}

// startGCLoop triggers the active expiration mechanism
func (e *Engine) startGCLoop() {
	ticker := time.NewTicker(e.cfg.GC.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := (*e.storage).DeleteExpired(e.cfg.GC.SamplesPerCheck)

			if stats > 0 {
				e.logger.Debug("GC delete expired", zap.Float64("expired_ratio", stats))
			}

			if stats < e.cfg.GC.MatchThreshold {
				break
			}
		case <-e.stopGC:
			e.logger.Info("GC stopped")
			return
		}
	}
}

// close signals background processes to shut down
func (e *Engine) close() {
	if e.cfg.GC.Enabled {
		close(e.stopGC)
	}
}

// register adds a new command to the engine. The command name is uppercase
func (e *Engine) register(name string, cmd command) {
	e.commands[strings.ToUpper(name)] = cmd
}

// registerBasicCommand fills the registry with standard commands
func (e *Engine) registerBasicCommand() {
	e.register("GET", commandFunc(get))
	e.register("SET", commandFunc(set))
	e.register("DEL", commandFunc(del))
	e.register("PING", commandFunc(ping))
	e.register("COMMAND", commandFunc(cmd))
	e.register("TTL", commandFunc(ttl))
	e.register("PTTL", commandFunc(pttl))
	e.register("PERSIST", commandFunc(persist))

	e.register("SAVE", commandFunc(func(ctx *context) resp.Value {
		if e.rdb == nil {
			return resp.MakeError("RDB disabled")
		}
		if err := e.rdb.Save(*e.storage); err != nil {
			return resp.MakeError(err.Error())
		}
		return resp.MakeSimpleString("OK")
	}))

	e.register("BGSAVE", commandFunc(func(ctx *context) resp.Value {
		if e.rdb == nil {
			return resp.MakeError("RDB disabled")
		}
		go func() {
			e.rdb.Save(*e.storage)
		}()
		return resp.MakeSimpleString("Background saving started")
	}))
}

// Execute finds the command by name and executes it with the passed arguments.
// If the command is not found, returns an error in the RESP format
func (e *Engine) Execute(name string, args []resp.Value) resp.Value {
	if e.logger.Core().Enabled(zap.DebugLevel) {
		// Log the command name and number of args
		e.logger.Debug("executing command",
			zap.String("cmd", name),
			zap.Int("args_count", len(args)),
		)
	}

	cmd, ok := e.commands[name]
	if !ok {
		return resp.MakeError(fmt.Sprintf("wrong command: %s", name))
	}

	ctx := &context{
		args:    args,
		storage: e.storage,
	}

	res := cmd.execute(ctx)

	if e.aof != nil && res.Type != resp.TypeError && isWriteCommand(name) {
		payload, err := resp.SerializeCommand(name, args)
		if err != nil {
			e.logger.Error("Failed to serialize command for AOF", zap.Error(err))
		} else {
			e.aof.Write(payload)
		}
	}

	return res
}

// Shutdown shuts down the engine and its background services correctly
func (e *Engine) Shutdown() {
	e.stopOnce.Do(func() {
		e.close()
		e.logger.Info("GC background process stopped")

		if e.aof != nil {
			e.aof.Close() //nolint:errcheck
		}
	})
}

// isWriteCommand helper what command change state database
func isWriteCommand(name string) bool {
	switch name {
	case "SET", "DEL", "PERSIST":
		return true
	}
	return false
}
