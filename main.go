package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	ReturnInvalidArguments  = 1
	ReturnConflictingArgs   = 2
	ReturnRuntimeError      = 3
	ReturnParseError        = 4
	ReturnIncorrectPassword = 5
	ReturnHostKeyUnknown    = 6
	ReturnHostKeyChanged    = 7
)

const version = "sshpass 1.10 (Go rewrite)"

func showHelp() {
	fmt.Printf(`Usage: sshpass [-f|-d|-p|-e[env_var]] [-hV] command parameters
   -f filename   Take password to use from file.
   -d number     Use number as file descriptor for getting password.
   -p password   Provide password as argument (security unwise).
   -e[env_var]   Password is passed as env-var "env_var" if given, "SSHPASS" otherwise.
   With no parameters - password will be taken from stdin.

   -P prompt     Which string should sshpass search for to detect a password prompt.
   -v            Be verbose about what you're doing.
   -h            Show help (this screen).
   -V            Print version information.
At most one of -f, -d, -p or -e should be used.
`)
}

func showVersion() {
	fmt.Printf(`%s
(C) 2006-2011 Lingnu Open Source Consulting Ltd.
(C) 2015-2016, 2021-2022 Shachar Shemesh
Go rewrite
This program is free software, and can be distributed under the terms of the GPL

Using "assword" as the default password prompt indicator.
`, version)
}

func parseOptions(args []string) (*Config, int, int) {
	config := &Config{SourceType: PWStdin}

	i := 1
outer:
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "--" || arg == "-" {
			break
		}

		for _, ch := range arg[1:] {
			switch ch {
			case 'f':
				if config.SourceType != PWStdin {
					fmt.Fprintf(os.Stderr, "Conflicting password source\n")
					return nil, 0, ReturnConflictingArgs
				}
				i++
				if i >= len(args) {
					return nil, 0, ReturnInvalidArguments
				}
				config.SourceType = PWFile
				config.File = args[i]
			case 'd':
				if config.SourceType != PWStdin {
					fmt.Fprintf(os.Stderr, "Conflicting password source\n")
					return nil, 0, ReturnConflictingArgs
				}
				i++
				if i >= len(args) {
					return nil, 0, ReturnInvalidArguments
				}
				fd, err := strconv.Atoi(args[i])
				if err != nil {
					return nil, 0, ReturnInvalidArguments
				}
				config.SourceType = PWFD
				config.FD = fd
			case 'p':
				if config.SourceType != PWStdin {
					fmt.Fprintf(os.Stderr, "Conflicting password source\n")
					return nil, 0, ReturnConflictingArgs
				}
				i++
				if i >= len(args) {
					return nil, 0, ReturnInvalidArguments
				}
				config.SourceType = PWPassword
				config.Password = args[i]
			case 'e':
				if config.SourceType != PWStdin {
					fmt.Fprintf(os.Stderr, "Conflicting password source\n")
					return nil, 0, ReturnConflictingArgs
				}
				envVar := "SSHPASS"
				if len(arg) > 2 {
					envVar = arg[2:]
				}
				config.SourceType = PWPassword
				pw := os.Getenv(envVar)
				if pw == "" {
					fmt.Fprintf(os.Stderr, "sshpass: -e option given but %q environment variable is not set.\n", envVar)
					return nil, 0, ReturnInvalidArguments
				}
				config.Password = pw
				os.Unsetenv(envVar)
				i++
				continue outer
			case 'P':
				i++
				if i >= len(args) {
					return nil, 0, ReturnInvalidArguments
				}
				config.Prompt = args[i]
			case 'v':
				config.Verbose++
			case 'h':
				showHelp()
				return nil, 0, -1
			case 'V':
				showVersion()
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "sshpass: invalid option -- %c\n", ch)
				return nil, 0, ReturnInvalidArguments
			}
		}
		i++
	}

	return config, i, 0
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	config, offset, errCode := parseOptions(os.Args)

	if errCode == -1 {
		os.Exit(0)
	}

	if errCode > 0 {
		fmt.Fprintf(os.Stderr, "Use \"sshpass -h\" to get help\n")
		os.Exit(errCode)
	}

	if offset >= len(os.Args) {
		showHelp()
		os.Exit(0)
	}

	ret := RunProgram(config, os.Args[offset:])
	os.Exit(ret)
}
