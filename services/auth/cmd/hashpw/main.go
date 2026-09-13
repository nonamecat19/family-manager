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

func readPassword(r io.Reader, prompt io.Writer) (string, error) {
	if f, ok := r.(*os.File); ok && isTerminal(f) {
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

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
