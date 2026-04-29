package main

import (
	"context"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
