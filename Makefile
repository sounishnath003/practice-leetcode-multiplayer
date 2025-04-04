DockerImageName="asia-south1-docker.pkg.dev/sounish-cloud-workstation/sounish-cloud-workstation/practice-leetcode-multiplayer"

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
	docker rmi -f $$(docker images -qa $(DockerImageName))
	docker build -t $(DockerImageName):$(date +'%Y.%m.%d') -f Dockerfile .