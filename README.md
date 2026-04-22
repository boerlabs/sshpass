# sshpass (Go rewrite)

A Go rewrite of [sshpass](https://sourceforge.net/projects/sshpass/) — a utility for non-interactive SSH password authentication.

It creates a pseudo-terminal (PTY), runs SSH as a child process, monitors the PTY output for password prompts, and supplies the password programmatically.

## Why

The original sshpass is written in C (~614 lines, single `main.c`). This rewrite:

- Provides the same CLI interface and behavior
- Has no external C dependencies — single static binary
- Handles edge cases the C version doesn't (e.g. segfaults on missing env vars)

## Install

```bash
go install github.com/boerlabs/sshpass@latest
```

Or build from source:

```bash
git clone https://github.com/boerlabs/sshpass.git
cd sshpass
go build -o sshpass .
```

## Usage

```
Usage: sshpass [-f|-d|-p|-e[env_var]] [-hV] command parameters
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
```

### Examples

```bash
# Password via command line (security unwise — visible in ps)
sshpass -p 'mypassword' ssh user@host command

# Password via environment variable
SSHPASS='mypassword' sshpass -e ssh user@host command

# Password via custom env var
MYPASS='mypassword' sshpass -eMYPASS ssh user@host command

# Password from file
echo 'mypassword' > ~/.ssh/pass
sshpass -f ~/.ssh/pass ssh user@host command

# Password from file descriptor
echo 'mypassword' | sshpass -d 0 ssh user@host command

# Password from stdin
echo 'mypassword' | sshpass ssh user@host command

# With custom port
sshpass -p 'mypassword' ssh -p 2222 user@host command
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Invalid arguments |
| 2 | Conflicting password sources |
| 3 | Runtime error |
| 4 | Parse error (reserved) |
| 5 | Incorrect password |
| 6 | Unknown host key |
| 7 | Changed host key |

## Security Considerations

- **`-p`** passes the password via command line arguments, which are visible to other users via `ps`. Avoid in production.
- **`-e`** reads from an environment variable and unsets it after reading. On Linux, the variable may still be briefly visible in `/proc/self/environ`.
- **`-d`** (file descriptor) is the most secure option. Use with process substitution or pipes.
- **`-f`** reads the password from a file — ensure proper file permissions (`chmod 600`).

## Compatibility

Linux only. Requires `/dev/ptmx` and PTY ioctl support.

## License

GPL — same as the original sshpass.
