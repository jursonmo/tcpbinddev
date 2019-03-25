package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	listenTcp()
}

func listenTcp() {
	ln, err := net.Listen("tcp", ":8090")
	if err != nil {
		panic(err)
	}
	fmt.Println("accept now......Listen :8090.............")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		_, err := conn.Write([]byte("aaaaaaaa"))
		if err != nil {
			fmt.Println("Write err:", err)
			return
		}
		fmt.Println("write ok\n")
		time.Sleep(time.Second)
	}
}
