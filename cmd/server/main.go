package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/eternalApril/moonlight/internal/resp"
	"github.com/eternalApril/moonlight/internal/server"
	"github.com/eternalApril/moonlight/internal/store"
)

func handleConnection(conn net.Conn, db store.Storage) {
	peer := server.NewPeer(conn)
	defer peer.Close()

	engine := server.NewEngine(db)

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
	db, err := store.NewShardedMapStore(32)
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", "0.0.0.0:6380")
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on :6380...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		fmt.Println("Accept connection")

		go handleConnection(conn, db)
	}
}
