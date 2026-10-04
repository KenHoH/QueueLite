package main

import (
	"log"

	"QueueLite/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
