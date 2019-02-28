## bind device for tcp connect
### golang tcp dial will complete 3-way handshake, so bindToDev() don't work. you can use syscall.Socket(), syscall.BindToDevice(fd, device) to implement. but it not join netPoll of golang , syscall.Read(fd) will create a system thread to read in blocking. i want to put fd to netPoll. try to use poll.FD:

	type FD struct {
		// Lock sysfd and serialize access to Read and Write methods.
		fdmu fdMutex

		// System file descriptor. Immutable until Close.
		Sysfd int
		...
	}

but "internal/poll" is not-export.

golang provides net.FileConn can make it. 


## how to check:
1. ifconfig eth0 192.168.1.2/24 //here mask 255.255.255.0
2. ifconfig eth1 192.168.1.3/16 //here mask 255.255.0.0
3. ping 192.168.1.1, icmp packet will be send to eth0, because route mask longest match
4. ping 192.168.1.1 -I eth1, will send to eth1
5. ./tcpbinddev -addr 192.168.1.1:6666 -device eth1 //if tcp syn send out from eth1, it means bind device successfully

## this only works in linux for now
