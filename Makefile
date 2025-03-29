
.PHONY: install
install:
	go mod tidy
	go mod download
	go mod verify

build: 
	go build -o ./bin/practice_leetcode_multiplayer main.go

run: build
	./bin/practice_leetcode_multiplayer