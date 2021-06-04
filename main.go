package main

import (
	"context"
	"github.com/werbenhu/amq/server"
)

func main() {
	ctx := context.Background()
	server := server.NewServer(ctx)
	server.Start()
}
