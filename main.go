package main

import (
	"log"

	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/core"
	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/server"
	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/utils"
)

func main() {

	co := core.Core{
		Port: utils.GetNumberFromEnv("PORT", 3000),
		Lo:   log.Default(),
	}

	// Init the server
	srv := server.Server{
		Co: &co,
	}

	panic(srv.StartServer())
}
