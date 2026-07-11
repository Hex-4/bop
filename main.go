package main

import (
	"fmt"
	"os"

	"github.com/Hex-4/bop/cli"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		if len(os.Args) < 3 || os.Args[2] != "run" {
			fmt.Println("usage: bop server run")
			os.Exit(1)
		}
		runServer()
	case "init":
		cli.RunInit()
	default:
		os.Exit(1)
	}
}
