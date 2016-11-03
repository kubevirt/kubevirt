package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Sleeping")
		// Sleep for one minute
		time.Sleep(60000 * time.Millisecond)
	}
}
