DockerImageName="asia-south1-docker.pkg.dev/sounish-cloud-workstation/sounish-cloud-workstation/practice-leetcode-multiplayer"

.PHONY: install
install:
	go mod tidy
	go mod download
	go mod verify

.PHONY: build
build: 
	CGO_ENBALED=0 go build -o ./bin/practice_leetcode_multiplayer main.go

.PHONY: run
run: build
	./bin/practice_leetcode_multiplayer

.PHONY: docker-build
docker-build:
	docker rmi -f $$(docker images -qa $(DockerImageName))
	docker build -t $(DockerImageName):$$(date +'%Y.%m.%d') -f Dockerfile .

.PHONY: deploy-application
deploy-application: docker-build
	docker push $(DockerImageName):$$(date +'%Y.%m.%d')
	gcloud run deploy practice-leetcode-multiplayer --image $(DockerImageName):$$(date +'%Y.%m.%d') --region asia-south1 --allow-unauthenticated --platform managed --set-env-vars=CODE_RUNNER_ENGINE_API=https://code-execution-engine-797087556919.asia-south1.run.app --cpu=1 --memory=256Mi --min=0 --max-instances=3


.PHONY: deploy-code-runner-engine
deploy-code-runner-engine:
	gcloud run deploy code-execution-engine --source ./code-execution-engine --region asia-south1 --no-allow-unauthenticated --platform managed --concurrency=100 --port 8080 --cpu=1 --memory=128Mi --min=0 --max-instances=2