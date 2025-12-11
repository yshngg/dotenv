package main

import (
	"fmt"
	"time"
)

func main() {
	count := 0
	for {
		fmt.Printf("Count: %v\n", count)
		time.Sleep(time.Millisecond * 100)
		count++
	}
}
