GO ?= go
PROTOC ?= protoc

PROTOIMPORT = ../../..
PROTOBUF = ${GOPATH}/src/github.com/linushung/hermes/rocks/hermes
PROTOPATH = ${GOPATH}/src/github.com/linushung/hermes/cmd/server/grpcserver/api

.PHONY: proto install build profile hermes

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

doc: ## Init a web server and presents the documentation as a web page.
	godoc -http=:8787

proto: ## Generate go protobuf files from .proto, default in cmd/server/grpcserver/api folder
	${PROTOC} -I=${PROTOIMPORT} -I=${PROTOPATH} --go_out=plugins=grpc:${PROTOPATH} ${PROTOPATH}/*.proto

init: ## Initialise new module project in current directory
	${GO} mod init ${m}

clean: ## Add missing and remove unused modules
	${GO} mod tidy -v

build: ## Build go program
	${GO} build -gcflags "-m -m"

docker: ## Build hermes Docker image. Use "tag=" flag to specify image tag
	docker build -t hermes:${tag} -f build/Dockerfile .

hermes: build ## Run hermes program
	./hermes

########## Profiling ##########
# Ref: https://www.integralist.co.uk/posts/profiling-go/
# Supported Porfile:
# goroutine    - stack traces of all current goroutines
# heap         - a sampling of memory allocations of live objects
# allocs       - a sampling of all past memory allocations
# threadcreate - stack traces that led to the creation of new OS threads
# block        - stack traces that led to blocking on synchronization primitives
# mutex        - stack traces of holders of contended mutexes
checkP: ## Check profile argument
ifndef p
	$(error You must specify profile type via p flag 😓...)
endif

checkS: ## Check seconds argument
ifeq ($(p),$(filter $(p),profile trace))
ifndef s
	$(error You must specify profile period(seconds) via s flag 😓...)
endif
endif

profile: ## Download specified profile over HTTP
ifeq ($(p),$(filter $(p),profile trace))
	http :8081/debug/pprof/${p}?seconds=${s} > profile/${p}.profile
else
	http :8081/debug/pprof/${p} > profile/${p}.profile
endif

pprof: checkP checkS profile ## Open pprof profile in speedscope & Web UI. Use "p=" flag to specify above profiles
	speedscope profile/${p}.profile
	pprof -http=:8585 profile/${p}.profile
# Example: make pprof p=profile s=10

trace: checkP checkS profile ## Open trace profile in trace UI. Use "s=" flag to specify tracing period
	go tool trace profile/trace.profile
# Example: make trace p=trace s=10
