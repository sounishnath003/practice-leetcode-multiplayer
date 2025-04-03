
.PHONY: install
install:
	go mod tidy
	go mod download
	go mod verify

build: 
	CGO_ENBALED=0 go build -o ./bin/practice_leetcode_multiplayer main.go

run: build
	./bin/practice_leetcode_multiplayer

docker-build:
	docker rmi -f $$(docker images -qa leetcode-multiplayer)
	docker build -t leetcode-multiplayer -f Dockerfile .