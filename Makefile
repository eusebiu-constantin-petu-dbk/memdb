export GO111MODULE=on

GOPATH := $(shell go env GOPATH)
BIN_DIR := bin
LEADER_BIN := $(BIN_DIR)/leader
REPLICA_BIN := $(BIN_DIR)/replica
LOCAL_REPLICA_BIN := $(BIN_DIR)/localreplica
PERF_BIN := $(BIN_DIR)/perf
LEADER_PORT := 8080
REPLICA_PORTS := 8081 8082 8083
REPLICA_URLS := $(foreach port,$(REPLICA_PORTS),http://localhost:$(port))

.PHONY: all
all: build

.PHONY: build
build: leader replica local-replica

.PHONY: leader
leader:
	go build -o $(LEADER_BIN) cmd/leader/main.go

.PHONY: replica
replica:
	go build -o $(REPLICA_BIN) cmd/replica/main.go

.PHONY: local-replica
local-replica:
	go build -o $(LOCAL_REPLICA_BIN) cmd/localreplica/main.go

.PHONY: perf
perf:
	go build -o $(PERF_BIN) cmd/perf/main.go

.PHONY: run-leader
run-leader: leader
	$(LEADER_BIN) $(LEADER_PORT) $(REPLICA_URLS)

.PHONY: run-replicas
run-replicas: replica
	$(REPLICA_BIN) 8081 http://localhost:8080 &
	$(REPLICA_BIN) 8082 http://localhost:8080 &
	$(REPLICA_BIN) 8083 http://localhost:8080 &
	@wait

.PHONY: run-local-replicas
run-local-replicas: local-replica
	$(LOCAL_REPLICA_BIN) 8081 &
	$(LOCAL_REPLICA_BIN) 8082 &
	$(LOCAL_REPLICA_BIN) 8083 &
	@wait

.PHONY: start-servers
start-servers: build
	$(MAKE) run-leader &
	sleep 1
	$(MAKE) run-replicas &
	@echo "Servers started. Press Ctrl+C to stop."
	@trap '$(MAKE) stop-servers' INT
	@wait

.PHONY: stop-servers
stop-servers: 
	@echo "Stopping servers..."
	-pkill -f '$(LEADER_BIN) $(LEADER_PORT)'
	-pkill -f '$(REPLICA_BIN) 8081'
	-pkill -f '$(REPLICA_BIN) 8082'
	-pkill -f '$(REPLICA_BIN) 8083'
	@echo "Servers stopped."

.PHONY: start-servers-local-replicas
start-servers-local-replicas: build
	$(MAKE) run-leader &
	sleep 1
	$(MAKE) run-local-replicas &
	@echo "Servers started. Press Ctrl+C to stop."
	@trap '$(MAKE) stop-servers' INT
	@wait

.PHONY: stop-servers-local-replicas
stop-servers-local-replicas:
	@echo "Stopping servers..."
	-pkill -f '$(LEADER_BIN) $(LEADER_PORT)'
	-pkill -f '$(LOCAL_REPLICA_BIN) 8081'
	-pkill -f '$(LOCAL_REPLICA_BIN) 8082'
	-pkill -f '$(LOCAL_REPLICA_BIN) 8083'
	@echo "Servers stopped."

.PHONY: test
test:
	CGO_ENABLED=1 go test -v -race ./...

.PHONY: lint
lint: install-lint
	golangci-lint run ./...

.PHONY: install-lint
install-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.42.1; \
	fi

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	@echo "Cleaned up binaries."

.PHONY: run-perf
run-perf: perf build clean-local-db
	$(MAKE) start-servers
	sleep 2
	$(PERF_BIN) http://localhost:$(LEADER_PORT) $(shell echo $(REPLICA_URLS))
	$(MAKE) stop-servers

.PHONY: run-perf-local-replicas
run-perf-local-replicas: perf build clean-local-db
	$(MAKE) start-servers-local-replicas
	sleep 2
	$(PERF_BIN) http://localhost:$(LEADER_PORT) $(shell echo $(REPLICA_URLS))
	$(MAKE) stop-servers-local-replicas

.PHONY: clean-local-db
clean-local-db:
	rm -f tmp/memdb/wordcounts.db
	@echo "Cleaned up local database."