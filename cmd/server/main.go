package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
		peer.Close() //nolint:errcheck
		// log connection close
		if log.Core().Enabled(zap.DebugLevel) {
			log.Debug("client disconnected", zap.String("addr", conn.RemoteAddr().String()))
		}
	}()

	for {
		cmdValue, err := peer.ReadCommand()
		if err != nil {
			if err != io.EOF {
				log.Warn("read command failed", zap.Error(err))
			}
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
			log.Error("error writing response:", zap.Error(err))
			return
		}

		if peer.InputBuffered() == 0 {
			if err := peer.Flush(); err != nil {
				return
			}
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
		log.Error("cant initialize storage", zap.Error(err))
		return
	}

	engine, err := server.NewEngine(db, cfg, log)
	if err != nil {
		log.Error("cant initialize storage", zap.Error(err))
		return
	}

	address := net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Error("listener error", zap.Error(err))
		return
	}
	log.Info("listening on", zap.String("address", address))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Error("Accept error", zap.Error(err))
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				handleConnection(conn, engine, log)
			}()
		}
	}()

	<-ctx.Done()

	log.Info("Shutting down...")

	listener.Close() //nolint:errcheck
	engine.Shutdown()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("All connections closed gracefully")
	case <-shutdownCtx.Done():
		log.Warn("Shutdown timed out, forcing exit", zap.Duration("timeout", 5*time.Second))
	}

	log.Info("Moonlight stopped")
}
