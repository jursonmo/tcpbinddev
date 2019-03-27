package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

var (
	lnaddr = flag.String("lnaddr", "0.0.0.0:8090", "listen addr")
	proto  = flag.String("proto", "tcp", "tcp or tls")
	tlsSK  = flag.String("server.key", "./config/server.key", "tls server.key")
	tlsSP  = flag.String("server.pem", "./config/server.pem", "tls server.pem")
)

func main() {
	flag.Parse()
	ln := newListener(*proto, *lnaddr)
	for {
		fmt.Println("listenning...............", *proto, *lnaddr)
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	//b := make([]byte, 1024)
	for {
		time.Sleep(time.Second)
		n, err := conn.Write([]byte("bbbbbbb"))
		if err != nil {
			fmt.Println(conn.LocalAddr().String(), conn.RemoteAddr().String(), err.Error())
			conn.Close()
			return
		}
		fmt.Println(conn.LocalAddr().String(), conn.RemoteAddr().String(), "write ok:", n)
	}
}

func newListener(mode string, lnaddr string) net.Listener {
	var ln net.Listener
	var err error
	var cert tls.Certificate
	switch mode {
	case "tls":
		if *tlsSP == "" || *tlsSK == "" {
			log.Fatalln("tlsSP, *tlsSK can’t be nil")
		}
		cert, err = tls.LoadX509KeyPair(*tlsSP, *tlsSK)
		if err != nil {
			log.Fatalln(err, *tlsSP, *tlsSK)
		}
		tlsconf := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		ln, err = tls.Listen("tcp4", lnaddr, tlsconf)
	case "tcp":
		ln, err = net.Listen("tcp4", lnaddr)
	}

	if err != nil {
		log.Fatalln(err)
	}
	return ln
}
