package main

import (
	"context"
	"os"

	"github.com/werbenhu/amqtt/config"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/server"
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
