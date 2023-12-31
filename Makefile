.PHONY: build test tools export healthcheck run-mocknet build-mocknet stop-mocknet ps-mocknet reset-mocknet logs-mocknet openapi

# compiler flags
NOW=$(shell date +'%Y-%m-%d_%T')
COMMIT:=$(shell git log -1 --format='%H')
VERSION:=$(shell cat version)
TAG?=testnet
ldflags = -X gitlab.com/mayachain/mayanode/constants.Version=$(VERSION) \
		  -X gitlab.com/mayachain/mayanode/constants.GitCommit=$(COMMIT) \
		  -X gitlab.com/mayachain/mayanode/constants.BuildTime=${NOW} \
		  -X github.com/cosmos/cosmos-sdk/version.Name=MAYAChain \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=mayanode \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X github.com/cosmos/cosmos-sdk/version.BuildTags=$(TAG)

# golang settings
TEST_DIR?="./..."
BUILD_FLAGS := -ldflags '$(ldflags)' -tags ${TAG}
TEST_BUILD_FLAGS := -parallel=1 -tags=mocknet
GOBIN?=${GOPATH}/bin
BINARIES=./cmd/mayanode ./cmd/bifrost ./tools/generate

# image build settings
BRANCH?=$(shell git rev-parse --abbrev-ref HEAD | sed -e 's/master/mocknet/g')
GITREF=$(shell git rev-parse --short HEAD)
BUILDTAG?=$(shell git rev-parse --abbrev-ref HEAD | sed -e 's/master/mocknet/g;s/develop/mocknet/g;s/testnet-multichain/testnet/g')
ifdef CI_COMMIT_BRANCH # pull branch name from CI, if available
	BRANCH=$(shell echo ${CI_COMMIT_BRANCH} | sed 's/master/mocknet/g')
	BUILDTAG=$(shell echo ${CI_COMMIT_BRANCH} | sed -e 's/master/mocknet/g;s/develop/mocknet/g;s/testnet-multichain/testnet/g')
endif

all: lint install

# ------------------------------ Generate ------------------------------

SMOKE_PROTO_DIR=test/smoke/mayanode_proto

protob:
	@./scripts/protocgen.sh

protob-docker:
	@docker run --rm -v $(shell pwd):/app -w /app \
		registry.gitlab.com/mayachain/mayanode:builder-v4@sha256:121369778ff891e34a750876306d4ce89f5069d13959aa39a0186d54f584ed1a \
		make protob

smoke-protob:
	@echo "Install betterproto..."
	@pip3 install --upgrade markupsafe==2.0.1 betterproto[compiler]==2.0.0b4
	@rm -rf "${SMOKE_PROTO_DIR}"
	@mkdir -p "${SMOKE_PROTO_DIR}"
	@echo "Processing thornode proto files..."
	@protoc \
  	-I ./proto \
  	-I ./third_party/proto \
  	--python_betterproto_out="${SMOKE_PROTO_DIR}" \
  	$(shell find ./proto -path -prune -o -name '*.proto' -print0 | xargs -0)

smoke-protob-docker:
	@docker run --rm -v $(shell pwd):/app -w /app \
		registry.gitlab.com/mayachain/mayanode:builder-v4@sha256:121369778ff891e34a750876306d4ce89f5069d13959aa39a0186d54f584ed1a \
		sh -c 'make smoke-protob'

$(SMOKE_PROTO_DIR):
	@$(MAKE) smoke-protob-docker

openapi:
	@docker run --rm \
		--user $(shell id -u):$(shell id -g) \
		-v $$PWD/openapi:/mnt \
		openapitools/openapi-generator-cli:v6.0.0@sha256:310bd0353c11863c0e51e5cb46035c9e0778d4b9c6fe6a7fc8307b3b41997a35 \
		generate -i /mnt/openapi.yaml -g go -o /mnt/gen
	@rm openapi/gen/go.mod openapi/gen/go.sum

# ------------------------------ Build ------------------------------

build: protob
	go build ${BUILD_FLAGS} ${BINARIES}

install: protob
	go install ${BUILD_FLAGS} ${BINARIES}

tools:
	go install -tags ${TAG} ./tools/generate
	go install -tags ${TAG} ./tools/pubkey2address

# ------------------------------ Housekeeping ------------------------------

format:
	@git ls-files '*.go' | grep -v -e '^docs/' | xargs gofumpt -w

lint:
	@./scripts/lint.sh
	@go run tools/analyze/main.go ./common/... ./constants/... ./x/...
	@./scripts/trunk check --no-fix --upstream origin/develop

lint-ci:
	@./scripts/lint.sh
	@go run tools/analyze/main.go ./common/... ./constants/... ./x/...
	@./scripts/trunk check --all --no-progress --monitor=false

clean:
	rm -rf ~/.maya*
	rm -f ${GOBIN}/{generate,mayanode,bifrost}

# ------------------------------ Testing ------------------------------

test-coverage:
	@go test ${TEST_BUILD_FLAGS} -v -coverprofile=coverage.txt -covermode count ${TEST_DIR}
	sed -i '/\.pb\.go:/d' coverage.txt

coverage-report: test-coverage
	@go tool cover -html=coverage.txt

test-coverage-sum:
	@go run gotest.tools/gotestsum --junitfile report.xml --format testname -- ${TEST_BUILD_FLAGS} -v -coverprofile=coverage.txt -covermode count ${TEST_DIR}
	sed -i '/\.pb\.go:/d' coverage.txt
	@GOFLAGS='${TEST_BUILD_FLAGS}' go run github.com/boumenot/gocover-cobertura < coverage.txt > coverage.xml
	@go tool cover -func=coverage.txt
	@go tool cover -html=coverage.txt -o coverage.html

test:
	@CGO_ENABLED=0 go test ${TEST_BUILD_FLAGS} ${TEST_DIR}

test-race:
	@go test -race ${TEST_BUILD_FLAGS} ${TEST_DIR}

test-watch:
	@gow -c test ${TEST_BUILD_FLAGS} ${TEST_DIR}

test-sync-mainnet:
	@BUILDTAG=mainnet BRANCH=mainnet $(MAKE) docker-gitlab-build
	@docker run --rm -e CHAIN_ID=mayachain-mainnet-v1 -e NET=mainnet registry.gitlab.com/mayachain/mayanode:mainnet

# ------------------------------ Docker Build ------------------------------

docker-gitlab-login:
	docker login -u ${CI_REGISTRY_USER} -p ${CI_REGISTRY_PASSWORD} ${CI_REGISTRY}

docker-gitlab-push:
	./build/docker/semver_tags.sh registry.gitlab.com/mayachain/mayanode ${BRANCH} $(shell cat version) \
		| xargs -n1 | grep registry | xargs -n1 docker push
	docker push registry.gitlab.com/mayachain/mayanode:${GITREF}

docker-gitlab-build:
	docker build -f build/docker/Dockerfile \
		$(shell sh ./build/docker/semver_tags.sh registry.gitlab.com/mayachain/mayanode ${BRANCH} $(shell cat version)) \
		-t registry.gitlab.com/mayachain/mayanode:${GITREF} --build-arg TAG=${BUILDTAG} .

# ------------------------------ Smoke Tests ------------------------------

SMOKE_DOCKER_OPTS = --network=host --rm -e RUNE=MAYA.CACAO -e LOGLEVEL=INFO -e PYTHONPATH=/app -w /app -v ${PWD}/test/smoke:/app

smoke-unit-test:
	@docker run ${SMOKE_DOCKER_OPTS} \
		-e EXPORT=${EXPORT} \
		-e EXPORT_EVENTS=${EXPORT_EVENTS} \
		registry.gitlab.com/mayachain/mayanode:smoke \
		sh -c 'python -m unittest tests/test_*'

smoke-build-image:
	@docker buildx build \
		-f test/smoke/Dockerfile -t registry.gitlab.com/mayachain/mayanode:smoke \
		./test/smoke

smoke-push-image:
	@docker push registry.gitlab.com/mayachain/mayanode:smoke

smoke: reset-mocknet smoke-build-image
	@docker run ${SMOKE_DOCKER_OPTS} \
		-e BLOCK_SCANNER_BACKOFF=${BLOCK_SCANNER_BACKOFF} \
		-v ${PWD}/test/smoke:/app \
		registry.gitlab.com/mayachain/mayanode:smoke \
		python scripts/smoke.py --fast-fail=True

smoke-remote-ci: reset-mocknet
	@docker run ${SMOKE_DOCKER_OPTS} \
		-e BLOCK_SCANNER_BACKOFF=${BLOCK_SCANNER_BACKOFF} \
		registry.gitlab.com/mayachain/mayanode:smoke \
		python scripts/smoke.py --fast-fail=True

# ------------------------------ Single Node Mocknet ------------------------------

cli-mocknet:
	@docker compose -f build/docker/docker-compose.yml run --rm cli

run-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet --profile midgard up -d

stop-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet --profile midgard down -v

build-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet --profile midgard build

bootstrap-mocknet: $(SMOKE_PROTO_DIR)
	@docker run ${SMOKE_DOCKER_OPTS} \
		-e BLOCK_SCANNER_BACKOFF=${BLOCK_SCANNER_BACKOFF} \
		-v ${PWD}/test/smoke:/app \
		registry.gitlab.com/mayachain/mayanode:smoke \
		python scripts/smoke.py --bootstrap-only=True

ps-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet --profile midgard images
	@docker compose -f build/docker/docker-compose.yml --profile mocknet --profile midgard ps

logs-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile midgard logs -f mayanode bifrost

reset-mayanode-mocknet: stop-mayanode-mocknet build-mayanode-mocknet run-mayanode-mocknet

build-mayanode-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mayanode build --no-cache

stop-mayanode-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mayanode stop

run-mayanode-mocknet:
	@docker compose -f build/docker/docker-compose.yml --profile mayanode up -d
 
reset-mocknet: stop-mocknet build-mocknet run-mocknet

restart-mocknet: stop-mocknet run-mocknet

# ------------------------------ Multi Node Mocknet ------------------------------

run-mocknet-cluster:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet-cluster --profile midgard up -d

stop-mocknet-cluster:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet-cluster --profile midgard down -v

build-mocknet-cluster:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet-cluster --profile midgard build --no-cache

ps-mocknet-cluster:
	@docker compose -f build/docker/docker-compose.yml --profile mocknet-cluster --profile midgard images
	@docker compose -f build/docker/docker-compose.yml --profile mocknet-cluster --profile midgard ps

reset-mocknet-cluster: stop-mocknet-cluster build-mocknet-cluster run-mocknet-cluster

update-thornode:
	@./scripts/update-thornode.sh ${THORCHAIN_VERSION}

rename:
	@./scripts/rename.sh
