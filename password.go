package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type PasswordSource int

const (
	PWStdin PasswordSource = iota
	PWFile
	PWFD
	PWPassword
)

type Config struct {
	SourceType PasswordSource
	File       string // -f
	FD         int    // -d
	Password   string // -p or -e
	Prompt     string // -P
	Verbose    int    // -v count
}

func (c *Config) WritePassword(dst io.Writer) error {
	switch c.SourceType {
	case PWStdin:
		return writePassFromReader(os.Stdin, dst)
	case PWFD:
		f := os.NewFile(uintptr(c.FD), "")
		if f == nil {
			return fmt.Errorf("invalid file descriptor %d", c.FD)
		}
		return writePassFromReader(f, dst)
	case PWFile:
		f, err := os.Open(c.File)
		if err != nil {
			return fmt.Errorf("failed to open password file %q: %w", c.File, err)
		}
		defer f.Close()
		return writePassFromReader(f, dst)
	case PWPassword:
		return writeLine(dst, c.Password)
	}
	return nil
}

func writeLine(dst io.Writer, content string) error {
	_, err := io.WriteString(dst, content+"\n")
	return err
}

func writePassFromReader(src io.Reader, dst io.Writer) error {
	reader := bufio.NewReader(src)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	line = strings.TrimRight(line, "\n\r")
	return writeLine(dst, line)
}
