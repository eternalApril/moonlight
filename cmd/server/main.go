package main

import (
	"net"
	"strings"

	"github.com/eternalApril/moonlight/internal/config"
	"github.com/eternalApril/moonlight/internal/logger"
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/server"
	"github.com/eternalApril/moonlight/internal/storage"
	"go.uber.org/zap"
)

// handleConnection handles a connection for a single user
func handleConnection(conn net.Conn, engine *server.Engine, log *zap.Logger) {
	if log.Core().Enabled(zap.DebugLevel) {
		log.Debug("client connected", zap.String("addr", conn.RemoteAddr().String()))
	}

	peer := server.NewPeer(conn)
	defer func() {
		peer.Close()
		// log connection close
		if log.Core().Enabled(zap.DebugLevel) {
			log.Debug("client disconnected", zap.String("addr", conn.RemoteAddr().String()))
		}
	}()

	for {
		cmdValue, err := peer.ReadCommand()
		if err != nil {
			return
		}

		if cmdValue.Type != resp.TypeArray {
			log.Error("invalid request type")
			continue
		}

		if len(cmdValue.Array) == 0 {
			continue
		}

		commandName := strings.ToUpper(string(cmdValue.Array[0].String))

		args := cmdValue.Array[1:]

		result := engine.Execute(commandName, args)

		if err = peer.Send(result); err != nil {
			log.Error("error writing response:", zap.String("error", err.Error()))
			return
		}
	}
}

func main() {
	cfg, err := config.Load(".")
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	defer log.Sync() //nolint:errcheck

	log.Info("Moonlight starting",
		zap.String("port", cfg.Server.Port),
		zap.Uint("shards", cfg.Storage.Shards),
	)

	db, err := storage.NewShardedMapStorage(cfg.Storage.Shards)
	if err != nil {
		log.Fatal("cant initialize storage", zap.String("error", err.Error()))
	}

	engine := server.NewEngine(db, cfg.GC, log)
	defer engine.Close()

	address := net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	log.Info("listening on", zap.String("address", address))

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("listener accept error", zap.String("error", err.Error()))
			continue
		}

		go handleConnection(conn, engine, log)
	}
}
