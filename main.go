package main

import (
	"crypto/tls"
	"fmt"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/server"
	"log"
	"net"
)

func main() {

	server := server.NewServer()

	tcpHost := config.TcpHost()
	var tcpListener net.Listener
	var err error

	if !config.IsTcpTsl() {
		tcpListener, err = net.Listen("tcp", tcpHost)
		if err != nil {
			log.Fatalf("tcp listen to %s Err:%s\n", tcpHost, err)
		}
	} else {
		cert, err := tls.LoadX509KeyPair(config.CaFile(), config.CeKey())
		if err != nil {
			log.Fatalf("tcp LoadX509KeyPair ce file: %s Err:%s\n", config.CaFile(), err)
		}
		tcpListener, err = tls.Listen("tcp", tcpHost, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		if err != nil {
			log.Fatalf("tsl listen to %s Err:%s\n", tcpHost, err)
		}
	}

	for {
		conn, err := tcpListener.Accept()
		ip := conn.RemoteAddr().String()
		fmt.Printf("tcp ip: %s\n", ip)

		if err != nil {
			log.Fatalf("tcp Accept to %s Err:%s\n", tcpHost, err)
			continue
		} else {
			go server.Handler(conn)
		}
	}
}
