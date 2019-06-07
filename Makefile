# all: compileGO compileJS compilePY

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

## ----------------------
## ------ Build
## ----------------------

build-environment: ## build the server executable (for linux/docker use only)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/environment ./cmd/environment
# build-mac: ## Build distribution binary for mac
# 	go build -o ./bin/simulation-osx ./cmd/server
# build-linux: ## Build distribution binary for linux
# 	GOOS=linux go build -a -installsuffix cgo -o ./bin/simulation-linux ./cmd/server

## ----------------------
## ------ Run
## ----------------------

run-e-testing: ## run the server locally with env set to testing
	go run -race ./cmd/environment/main.go -grpc-port=9091 -log-level=-1 -env=testing
run-e-training: ## run the server locally with env set to training
	go run -race ./cmd/environment/main.go -grpc-port=9091 -log-level=-1 -env=training
run-e-prod: ## run the server locally with env set to prod
	go run -race ./cmd/environment/main.go -grpc-port=9091 -log-level=-1 -env=prod

run-c-testing: ## run the server locally with env set to testing
	go run -race ./cmd/collective/main.go -grpc-port=9090 -redis-addr=localhost:6379 -environment-addr=localhost:9091 -log-level=-1 -env=testing
run-c-training: ## run the server locally with env set to training
	go run -race ./cmd/collective/main.go -grpc-port=9090 -log-level=-1 -env=training
run-c-prod: ## run the server locally with env set to prod
	go run -race ./cmd/collective/main.go -grpc-port=9090 -log-level=-1 -env=prod

## ----------------------
## ------ Testing
## ----------------------

test: test-vec2 test-simulation ## test all internal packages
test-vec2: ## run tests for Vec2
	go test ./pkg/vec2/v1
test-simulation: ## run tests for the simulation service
	go test ./pkg/service/v1

## ----------------------
## ------ Protobuf
## ----------------------

compile-proto: compile-proto-go compile-proto-py compile-proto-js # compile proto in all languages
compile-proto-go:
	./third_party/protoc-gen-go.sh
compile-proto-py:
	./third_party/protoc-gen-py.sh
compile-proto-js:
	./third_party/protoc-gen-js.sh

## ----------------------
## ------ DOCKER 
## ----------------------

check-version-env-var:
ifndef VERSION
	$(error VERSION is undefined)
endif

docker-build-environment: check-version-env-var ## build the docker image, must have variable VERSION
	docker build -t terrariumai/environment:$(VERSION) -f ./docker/environment/Dockerfile .

# Pushing the docker builds
docker-push-environment: check-version-env-var ## push the docker image
	docker push terrariumai/environment:$(VERSION)

dockerize-environment: build-environment docker-build-environment docker-push-environment ## build and push dev proxy

	
