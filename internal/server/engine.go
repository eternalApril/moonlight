package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/eternalApril/moonlight/internal/config"
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/storage"
	"go.uber.org/zap"
)

type Engine struct {
	commands map[string]Command
	storage  *storage.Storage
	gcConf   config.GCConfig
	stopGC   chan struct{}
	logger   *zap.Logger
}

func NewEngine(s storage.Storage, gcConf config.GCConfig, logger *zap.Logger) *Engine {
	engine := Engine{
		commands: make(map[string]Command),
		storage:  &s,
		gcConf:   gcConf,
		stopGC:   make(chan struct{}),
		logger:   logger,
	}
	engine.registerBasicCommand()

	if gcConf.Enabled {
		go engine.startGCLoop()
	}

	return &engine
}

// startGCLoop triggers the active expiration mechanism
func (e *Engine) startGCLoop() {
	ticker := time.NewTicker(e.gcConf.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := (*e.storage).DeleteExpired(e.gcConf.SamplesPerCheck)

			if stats > 0 {
				e.logger.Debug("GC delete expired", zap.Float64("expired_ratio", stats))
			}

			if stats < e.gcConf.MatchThreshold {
				break
			}
		case <-e.stopGC:
			e.logger.Info("GC stopped")
			return
		}
	}
}

func (e *Engine) Close() {
	if e.gcConf.Enabled {
		close(e.stopGC)
	}
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
	e.register("PERSIST", CommandFunc(persist))
}

func (e *Engine) Execute(name string, args []resp.Value) resp.Value {
	if e.logger.Core().Enabled(zap.DebugLevel) {
		// Log the command name and number of args.
		e.logger.Debug("executing command",
			zap.String("cmd", name),
			zap.Int("args_count", len(args)),
		)
	}

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
