// +build darwin

package tcpbinddev

import (
	"fmt"
	"math"
	"net"
	"syscall"
)

// Create new socket passing (domain, typ, proto) to socket syscall
// set FD_CLOEXEC socket's file descriptor flag and O_NONBLOCK
// file status flag
func newSocketCloexec(domain, typ, proto int) (int, error) {
	syscall.ForkLock.RLock()
	fd, err := syscall.Socket(domain, typ, proto)
	if err == nil {
		syscall.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return -1, fmt.Errorf("cannot create listening socket: %w", err)
	}
	if err = syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return -1, fmt.Errorf("cannot make non-blocked listening socket: %w", err)
	}
	return fd, nil
}

// Block until fd will be ready for write or time specified by seconds will pass
func connectTimeout(fd, seconds int) error {
	w := &syscall.FdSet{}
	if fd > math.MaxInt32 {
		return fmt.Errorf("cannot convert fd=%d to int32", fd)
	}
	w.Bits[0] = int32(fd)
	err := syscall.Select(fd+1, nil, w, nil, &syscall.Timeval{Sec: int64(seconds)})
	if err != nil {
		return fmt.Errorf("cannot select fd=%d: %w", fd, err)
	}
	v, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
	if err != nil {
		return fmt.Errorf("cannot get SO_ERROR socket options: %w", err)
	} else if v != 0 {
		return fmt.Errorf("there is error on the socket: %d", v)
	}
	return nil
}

// Bound socket defined with fd to ifaceName
func bindToInterface(fd int, ifaceName string) error {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_BOUND_IF, iface.Index)
	if err != nil {
		return err
	}
	return nil
}
