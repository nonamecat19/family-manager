// Command hashpw prints an argon2id hash for a password, using the same parameters the auth
// service uses at runtime.
//
// It exists for the two jobs that otherwise end in someone pasting a hash from the internet:
// seeding a first account into a fresh database, and resetting a password when nobody can
// sign in to do it the normal way.
//
//	just hashpw
//	go run ./cmd/hashpw < /dev/tty
//
// The password is read from stdin, not from a flag or an argument. An argument would put the
// password in the shell history, in `ps` output for the duration of the run, and in any
// process listing a shared machine keeps.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nnc/family-manager/services/auth/internal/password"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "hashpw:", err)
		os.Exit(1)
	}
}

func run() error {
	p := password.DefaultParams()
	memory := flag.Uint("memory", uint(p.Memory), "argon2id memory in KiB")
	iterations := flag.Uint("iterations", uint(p.Iterations), "argon2id iterations")
	parallelism := flag.Uint("parallelism", uint(p.Parallelism), "argon2id lanes")
	flag.Parse()

	p.Memory = uint32(*memory)
	p.Iterations = uint32(*iterations)
	p.Parallelism = uint8(*parallelism)

	plaintext, err := readPassword(os.Stdin, os.Stderr)
	if err != nil {
		return err
	}

	hash, err := password.Hash(plaintext, p)
	if err != nil {
		return err
	}
	fmt.Println(hash)
	return nil
}

// readPassword takes the first line of r. A trailing newline is stripped and nothing else is:
// a password may legitimately start or end with a space, so trimming would silently hash
// something the user did not type.
func readPassword(r io.Reader, prompt io.Writer) (string, error) {
	if f, ok := r.(*os.File); ok && isTerminal(f) {
		// A prompt that fails to print is not a reason to refuse to hash.
		_, _ = fmt.Fprint(prompt, "password (input is echoed): ")
	}

	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && (err != io.EOF || line == "") {
		return "", fmt.Errorf("read password: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", fmt.Errorf("empty password")
	}
	return line, nil
}

// isTerminal reports whether f is a character device, which is enough to decide whether to
// print a prompt. Anything more would mean a terminal library for one line of output.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
