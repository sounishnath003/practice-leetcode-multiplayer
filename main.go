package main

import "log"

func main() {
	for i := 0; i < 10; i++ {
		log.Println("Hello world!", "value", i)
	}
}
