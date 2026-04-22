package main

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	ctrlC = 0x03
	ctrlZ = 0x1a
)

type SignalHandler struct {
	masterPt *os.File
	ourTty   *os.File
	childPid int
	sigCh    chan os.Signal
}

func NewSignalHandler(masterPt *os.File, childPid int) *SignalHandler {
	return &SignalHandler{
		masterPt: masterPt,
		childPid: childPid,
		sigCh:    make(chan os.Signal, 1),
	}
}

func (s *SignalHandler) Start() <-chan os.Signal {
	if f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0); err == nil {
		s.ourTty = f
	}

	signal.Notify(s.sigCh,
		syscall.SIGWINCH,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTSTP,
	)
	return s.sigCh
}

func (s *SignalHandler) Handle(sig os.Signal) {
	switch sig {
	case syscall.SIGWINCH:
		s.propagateWindowSize()
	case syscall.SIGINT:
		s.masterPt.Write([]byte{ctrlC})
	case syscall.SIGTSTP:
		s.masterPt.Write([]byte{ctrlZ})
	case syscall.SIGTERM, syscall.SIGHUP:
		if s.childPid > 0 {
			syscall.Kill(s.childPid, sig.(syscall.Signal))
		}
	}
}

func (s *SignalHandler) propagateWindowSize() {
	if s.ourTty == nil || s.masterPt == nil {
		return
	}
	ws, err := unix.IoctlGetWinsize(int(s.ourTty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return
	}
	unix.IoctlSetWinsize(int(s.masterPt.Fd()), unix.TIOCSWINSZ, ws)
}

func propagateWindowSizeOnce(dstFd int) {
	ttyFd, err := unix.Open("/dev/tty", unix.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer unix.Close(ttyFd)
	ws, err := unix.IoctlGetWinsize(ttyFd, unix.TIOCGWINSZ)
	if err != nil {
		return
	}
	unix.IoctlSetWinsize(dstFd, unix.TIOCSWINSZ, ws)
}

func (s *SignalHandler) Stop() {
	signal.Stop(s.sigCh)
	if s.ourTty != nil {
		s.ourTty.Close()
	}
}
