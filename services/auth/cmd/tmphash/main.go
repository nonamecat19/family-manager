package main

import (
	"fmt"
	"os"

	"github.com/nnc/family-manager/services/auth/internal/password"
)

func main() {
	h, err := password.Hash(os.Args[1], password.DefaultParams())
	if err != nil {
		panic(err)
	}
	fmt.Println(h)
}
