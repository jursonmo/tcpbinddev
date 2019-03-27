package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/jursonmo/tcpbinddev"
)

/*
testting example:
	ifconfig eth0 192.168.1.2/24 //here mask 255.255.255.0
	ifconfig eth1 192.168.1.3/16 //here mask 255.255.0.0
	ping 192.168.1.1 will send to eth0, because route mask longest match
	ping 192.168.1.1 -I eth1, will send to eth1
	./tcpbinddev -addr 192.168.1.1:6666 -device eth1 //if tcp syn send out from eth1, it means bind device successfully
*/
func main() {
	var addr, device, proto string
	flag.StringVar(&addr, "addr", "127.0.0.1:8090", "dst addr")
	flag.StringVar(&addr, "saddr", "", "src addr, like x.x.x.x:8090")
	flag.StringVar(&device, "device", "", "bind device")
	flag.StringVar(&proto, "proto", "tcp", "tcp or tls")
	flag.Parse()
	pn := 1
	old := runtime.GOMAXPROCS(pn)
	fmt.Printf("old OMAXPROCS:%d, set to %d\n", old, pn)
	network := "tcp4"
	dialTimeout := 3 //seconds
	var wg sync.WaitGroup
	f := func(index int) {
		defer wg.Done()
		var conn net.Conn
		var err error
		if proto == "tls" {
			tlsconf := &tls.Config{
				InsecureSkipVerify: true,
			}
			conn, err = tcpbinddev.TlsBindToDev(network, addr, "", device, dialTimeout, tlsconf)
		} else {
			conn, err = tcpbinddev.TcpBindToDev(network, addr, "", device, dialTimeout)
		}

		if err != nil {
			fmt.Println("main,TcpBindToDev", err)
			return
		}
		defer conn.Close()
		fmt.Printf("connect successfully: goroutine id=%d, localaddr:%s, remote:%s\n",
			index, conn.LocalAddr().String(), conn.RemoteAddr().String())

		time.Sleep(time.Second * 3) //wait for other goroutine connect success
		buf := make([]byte, 1024)
		i := 0
		for i < 3 {
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("read err:", err)
				return
			}
			i++
			fmt.Printf("goroutine id=%d, read buf:%s\n", index, string(buf[:n]))
		}
		fmt.Printf("goroutine id=%d, quit read \n", index)
	}
	//testing: if tcpbinddev put conn to netPoll successfully
	//there will be two goroutine run on one system thread
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go f(i)
	}
	wg.Wait()
	//check: lsof -p `pidof tcpbinddev`, make sure socket have closed, and no file descriptor leak
	time.Sleep(time.Hour)
}
