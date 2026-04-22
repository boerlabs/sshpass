package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	defaultPrompt        = "assword"
	hostAuthPrompt       = "The authenticity of host "
	hostKeyChangedPrompt = "differs from the key for the IP address"
)

func ptsname(fd int) (string, error) {
	var n uint32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), unix.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return "", errno
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

func RunProgram(config *Config, args []string) int {
	masterFd, err := unix.Open("/dev/ptmx", unix.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get a pseudo terminal: %v\n", err)
		return ReturnRuntimeError
	}
	masterPt := os.NewFile(uintptr(masterFd), "master_pt")

	var unlock int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(masterFd), unix.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock)))
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "Failed to unlock pseudo terminal: %v\n", errno)
		masterPt.Close()
		return ReturnRuntimeError
	}

	slaveName, err := ptsname(masterFd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get slave PTY name: %v\n", err)
		masterPt.Close()
		return ReturnRuntimeError
	}

	propagateWindowSizeOnce(masterFd)

	slaveFd, err := unix.Open(slaveName, unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open slave PTY: %v\n", err)
		masterPt.Close()
		return ReturnRuntimeError
	}
	slavePt := os.NewFile(uintptr(slaveFd), "slave_pt")

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = slavePt
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "SSHPASS: Failed to run command: %v\n", err)
		masterPt.Close()
		slavePt.Close()
		return ReturnRuntimeError
	}

	childPid := cmd.Process.Pid

	sigHandler := NewSignalHandler(masterPt, childPid)
	sigCh := sigHandler.Start()

	type readResult struct {
		data []byte
		err  error
	}
	readCh := make(chan readResult, 16)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := masterPt.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				readCh <- readResult{data: data}
			}
			if err != nil {
				readCh <- readResult{err: err}
				return
			}
		}
	}()

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
	}()

	prompt := config.Prompt
	if prompt == "" {
		prompt = defaultPrompt
	}
	matcherPrompt := NewMatcher(prompt)
	matcherHostAuth := NewMatcher(hostAuthPrompt)
	matcherHostKey := NewMatcher(hostKeyChangedPrompt)

	if config.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "SSHPASS: searching for password prompt using match %q\n", prompt)
	}

	prevMatch := false
	terminate := 0
	var waitErr error
	childDone := false

mainLoop:
	for {
		if childDone && terminate != 0 {
			break
		}

		select {
		case sig := <-sigCh:
			sigHandler.Handle(sig)
		case result := <-readCh:
			if len(result.data) > 0 {
				if config.Verbose > 0 {
					fmt.Fprintf(os.Stderr, "SSHPASS: read: %s", result.data)
				}

				if terminate == 0 {
					matcherPrompt.Match(result.data)
					if matcherPrompt.Matched() {
						if !prevMatch {
							if config.Verbose > 0 {
								fmt.Fprintf(os.Stderr, "SSHPASS: detected prompt. Sending password.\n")
							}
							config.WritePassword(masterPt)
							matcherPrompt.Reset()
							prevMatch = true
						} else {
							if config.Verbose > 0 {
								fmt.Fprintf(os.Stderr, "SSHPASS: detected prompt, again. Wrong password. Terminating.\n")
							}
							terminate = ReturnIncorrectPassword
							syscall.Kill(childPid, syscall.SIGTERM)
						}
					}
				}

				if terminate == 0 {
					matcherHostAuth.Match(result.data)
					if matcherHostAuth.Matched() {
						if config.Verbose > 0 {
							fmt.Fprintf(os.Stderr, "SSHPASS: detected host authentication prompt. Exiting.\n")
						}
						terminate = ReturnHostKeyUnknown
					} else {
						matcherHostKey.Match(result.data)
						if matcherHostKey.Matched() {
							terminate = ReturnHostKeyChanged
						}
					}
				}
			}
			if result.err != nil {
				if !childDone {
					waitErr = <-exitCh
				}
				break mainLoop
			}
		case waitErr = <-exitCh:
			childDone = true
			slavePt.Close()
			if terminate != 0 {
				break mainLoop
			}
		}
	}

	masterPt.Close()
	slavePt.Close()
	sigHandler.Stop()

	if !childDone {
		waitErr = <-exitCh
	}

	if terminate > 0 {
		return terminate
	}
	return exitCode(waitErr)
}

func exitCode(waitErr error) int {
	if waitErr == nil {
		return 0
	}
	exitErr, ok := waitErr.(*exec.ExitError)
	if !ok {
		return ReturnRuntimeError
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return ReturnRuntimeError
	}
	if status.Exited() {
		return status.ExitStatus()
	}
	return 255
}
