BINARY := go-tcp-lb

# default port
LISTEN := :4000

# Default Backends
BACKENDS := 127.0.0.1:5001,127.0.0.1:5002

GO := go

.PHONY: build
build:
	@echo "Building $(BINARY)..."
	$(GO) build -o $(BINARY) main.go

.PHONY: run
run:
	@echo "Running $(BINARY)..."
	@LISTEN=$(LISTEN) BACKENDS=$(BACKENDS) $(GO) run main.go

.PHONY: run-server
run-server:
	@echo "Running server..."
	@$(GO) run tools/test_backend.go

.PHONY: run-bin
run-bin: build
	@echo "Running compiled $(BINARY)..."
	@LISTEN=$(LISTEN) BACKENDS=$(BACKENDS) ./$(BINARY)

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)

.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t go-tcp-lb .

.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -p 4000:4000 -e BACKENDS="$(BACKENDS)" go-tcp-lb

.PHONY: health
health:
	@echo "Testing if TCP proxy is alive..."
	@nc -vz 127.0.0.1 4000
