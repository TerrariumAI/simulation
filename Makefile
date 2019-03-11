# all: compileGO compileJS compilePY

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

go-build: ## build the server executable (for linux/docker use only)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

run: ## run the server locally
	go run ./cmd/server/main.go -grpc-port=9090 -http-port=8080 -log-level=-1 -env=prod
run-testing: ## run the server locally
	go run ./cmd/server/main.go -grpc-port=9090 -http-port=8080 -log-level=-1 -env=testing

check-version-env-var:
ifndef VERSION
	$(error VERSION is undefined)
endif

## ----------------------
## ------ DOCKER --------
## ----------------------
dockerize: check-version-env-var docker-build docker-push ## build and push dev proxy

# Building the docker builds
docker-build: check-version-env-var go-build ## build the docker image, must have variable VERSION
	docker build -t olamai/simulation:$(VERSION) -f ./Dockerfile .
	
# Pushing the docker builds
docker-push: check-version-env-var ## push the docker image
	docker push olamai/simulation:$(VERSION)