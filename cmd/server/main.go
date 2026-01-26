package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/eternalApril/moonlight/internal/config"
	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/server"
	"github.com/eternalApril/moonlight/internal/store"
)

func handleConnection(conn net.Conn, engine *server.Engine) {
	peer := server.NewPeer(conn)
	defer peer.Close()

	for {
		cmdValue, err := peer.ReadCommand()
		if err != nil {
			return
		}

		if cmdValue.Type != resp.TypeArray {
			fmt.Println("Invalid request type")
			continue
		}

		if len(cmdValue.Array) == 0 {
			continue
		}

		commandName := strings.ToUpper(string(cmdValue.Array[0].String))

		args := cmdValue.Array[1:]

		result := engine.Execute(commandName, args)

		if err := peer.Send(result); err != nil {
			fmt.Println("Error writing response:", err)
			return
		}
	}
}

func main() {
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Moonlight initializing... (Port: %s, Shards: %d, GC: %v)\n",
		cfg.Server.Port, cfg.Storage.Shards, cfg.GC.Enabled)

	db, err := store.NewShardedMapStore(cfg.Storage.Shards)
	if err != nil {
		panic(err)
	}

	engine := server.NewEngine(db, cfg.GC)
	defer engine.Close()

	address := net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Listening on %s...\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		go handleConnection(conn, engine)
	}
}
