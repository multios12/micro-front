package main

import (
	"context"
	"log"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.SetFlags(0)
		log.Println(err)
		os.Exit(1)
	}
}
