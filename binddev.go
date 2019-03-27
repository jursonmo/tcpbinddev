package tcpbinddev

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

//timeout is seconds
func TcpBindToDev(network, addr, saddr, device string, timeout int) (net.Conn, error) {
	if network == "" || addr == "" {
		return nil, errors.New("network or addr not set")
	}
	sa, soType, err := getSockaddr(network, addr)
	if err != nil {
		return nil, errors.Wrap(err, "getSockaddr->")
	}

	fd, err := newSocketCloexec(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return nil, errors.Wrap(err, "newSocketCloexec->")
	}
	fmt.Printf("fd:%d\n", fd)

	err = fdSetOpt(fd, network, saddr, device)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}
	err = syscall.Connect(fd, sa)
	if err != nil && err.(syscall.Errno) != syscall.EINPROGRESS {
		//EINPROGRESS: The socket is nonblocking and the  connection  cannot  be  completed immediately.
		syscall.Close(fd)
		return nil, errors.Wrap(err, "Connect->")
	}
	err = connectTimeout(fd, timeout)
	if err != nil {
		syscall.Close(fd)
		return nil, errors.Wrap(err, "connectTimeout->")
	}

	//now set nonblocking
	// err = syscall.SetNonblock(fd, true)
	// if err != nil {
	// 	syscall.Close(fd)
	// 	return nil, errors.Wrap(err, "SetNonblock->")
	// }

	name := "tcp socket to netPoll"
	file := os.NewFile(uintptr(fd), name)
	conn, err := net.FileConn(file)
	if err != nil {
		file.Close()
		return nil, errors.Wrap(err, "FileConn->")
	}

	//close file does not affect conn
	if err := file.Close(); err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "file.Close() error->")
	}
	return conn, nil
}

func getSockaddr(network, addr string) (sa syscall.Sockaddr, soType int, err error) {
	if network != "tcp4" && network != "tcp6" {
		return nil, -1, errors.New("only tcp4 and tcp6 network is supported")
	}

	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return nil, -1, err
	}

	switch network {
	case "tcp4":
		var sa4 syscall.SockaddrInet4
		sa4.Port = tcpAddr.Port
		copy(sa4.Addr[:], tcpAddr.IP.To4())
		return &sa4, syscall.AF_INET, nil
	case "tcp6":
		var sa6 syscall.SockaddrInet6
		sa6.Port = tcpAddr.Port
		copy(sa6.Addr[:], tcpAddr.IP.To16())
		if tcpAddr.Zone != "" {
			ifi, err := net.InterfaceByName(tcpAddr.Zone)
			if err != nil {
				return nil, -1, err
			}
			sa6.ZoneId = uint32(ifi.Index)
		}
		return &sa6, syscall.AF_INET6, nil
	default:
		return nil, -1, errors.New("Unknown network type " + network)
	}
}

func fdSetOpt(fd int, network, saddr string, device string) error {
	var err error

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("cannot enable SO_REUSEADDR: %s", err)
	}

	// This should disable Nagle's algorithm in all accepted sockets by default.
	// Users may enable it with net.TCPConn.SetNoDelay(false).
	if err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
		return fmt.Errorf("cannot disable Nagle's algorithm: %s", err)
	}

	if device != "" {
		err = syscall.BindToDevice(fd, device)
		if err != nil {
			return errors.Wrap(err, "BindToDevice->")
		}
	}

	if network != "" && saddr != "" {
		sa, _, err := getSockaddr(network, saddr)
		if err != nil {
			return errors.Wrap(err, "getSockaddr->")
		}
		err = syscall.Bind(fd, sa)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Bind Saddr:%s fail->", saddr))
		}
	}
	return nil
}
type timeoutError struct{}

func (timeoutError) Error() string   { return "tls: DialWithDialer timed out" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

//timeout is seconds
func TlsBindToDev(network, addr, saddr, device string, timeout int, config *tls.Config) (net.Conn, error) {
	colonPos := strings.LastIndex(addr, ":")
	if colonPos == -1 {
		colonPos = len(addr)
	}
	hostname := addr[:colonPos]

	if config == nil {
		return nil, fmt.Errorf("config is not set")
	}
	// If no ServerName is set, infer the ServerName
	// from the hostname we're connecting to.
	if config.ServerName == "" {
		// Make a copy to avoid polluting argument or default.
		c := config.Clone()
		c.ServerName = hostname
		config = c
	}

	var errChannel chan error

	if timeout != 0 {
		errChannel = make(chan error, 2)
		time.AfterFunc(time.Second*time.Duration(timeout), func() {
			errChannel <- timeoutError{}
		})
	}

	rawConn, err := TcpBindToDev(network, addr, saddr, device, timeout)
	if err != nil {
		return nil, err
	}
	conn := tls.Client(rawConn, config)

	if timeout == 0 {
		err = conn.Handshake()
	} else {
		go func() {
			errChannel <- conn.Handshake()
		}()

		err = <-errChannel
	}

	if err != nil {
		rawConn.Close()
		return nil, err
	}

	return conn, nil
}
