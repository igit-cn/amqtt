package main

import (
	"context"
	"os"

	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/logger"
	"github.com/werbenhu/amq/server"
)

func main() {
	err := config.Configure(os.Args[1:])
	if err != nil {
		logger.Fatal("configure config error: ", err)
	}
	ctx := context.Background()
	server := server.NewServer(ctx)
	server.Start()
}
