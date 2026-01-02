package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	mode := "all"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	fmt.Printf("sercha-core %s\n", version)
	fmt.Printf("mode: %s\n", mode)
	fmt.Println("ready to serve")
}
