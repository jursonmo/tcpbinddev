// +build !darwin

package tcpbinddev

import (
	"fmt"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

func newSocketCloexec(domain, typ, proto int) (int, error) {
	fd, err := syscall.Socket(domain, typ|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, proto)
	//fd, err := syscall.Socket(domain, typ|syscall.SOCK_CLOEXEC, proto)
	if err == nil {
		return fd, nil
	}

	if err == syscall.EPROTONOSUPPORT || err == syscall.EINVAL {
		return newSocketCloexecOld(domain, typ, proto)
	}

	return -1, fmt.Errorf("cannot create listening unblocked socket: %s", err)
}

func newSocketCloexecOld(domain, typ, proto int) (int, error) {
	syscall.ForkLock.RLock()
	fd, err := syscall.Socket(domain, typ, proto)
	if err == nil {
		syscall.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return -1, fmt.Errorf("cannot create listening socket: %s", err)
	}
	if err = syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return -1, fmt.Errorf("cannot make non-blocked listening socket: %s", err)
	}
	return fd, nil
}

// fd is noblock, select to check fd if ok
func connectTimeout(fd, seconds int) error {
	fmt.Println("Select for connect ......", time.Now())
	w := &FDSet{}
	w.Zero()
	w.Set(uintptr(fd))
	ret, e := syscall.Select(fd+1, nil, (*syscall.FdSet)(w), nil, &syscall.Timeval{Sec: int64(seconds)})
	fmt.Printf("Select over:ret=%d, time:%s\n", ret, time.Now())
	if e != nil {
		return errors.Wrap(e, "Select->")
	}
	if ret <= 0 {
		return errors.New("Select->")
	}
	v, _ := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
	if v != 0 {
		return errors.New("GetsockoptInt->")
	}
	return nil
}

/*
Situation: You set up a non-blocking socket and do a connect() that returns -1/EINPROGRESS or -1/EWOULDBLOCK.
 You select() the socket for writability. This returns as soon as the connection succeeds or fails.
 (Exception: Under some old versions of Ultrix, select() wouldn't notice failure before the 75-second timeout.)
Question: What do you do after select() returns writability? Did the connection fail? If so, how did it fail?
If the connection failed, the reason is hidden away inside something called so_error in the socket.
Modern systems let you see so_error with getsockopt(,,SO_ERROR,,) ...
*/
