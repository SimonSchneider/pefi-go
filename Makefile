PORT := 3002
CONTAINER_RUNTIME ?= podman
VERSION ?= main
EXTRA_TAGS ?=
REGISTRY ?= ghcr.io/simonschneider/pefi-go
IMAGE ?= $(REGISTRY):$(VERSION)

watch-tw:
	@echo "Watching for changes..."
	@./vendor-bin/tailwindcss -i tailwind/styles.css -o static/public/styles-tw.css --watch

watch-templ:
	@go tool templ generate --watch --proxy="http://localhost:$(PORT)" --cmd="go run cmd/main.go -addr :$(PORT) -watch -dburl tmp.db.sqlite"

watch-all:
	@echo "Watching Templ and Tailwind (Ctrl+C to stop both)..."
	@trap 'kill 0' EXIT; \
	./vendor-bin/tailwindcss -i tailwind/styles.css -o static/public/styles-tw.css --watch & \
	go tool templ generate --watch --proxy="http://localhost:$(PORT)" --cmd="go run cmd/main.go -addr :$(PORT) -watch -dburl tmp.db.sqlite"

generate:
	@echo "Generating code..."
	@go generate ./...
	@go tool templ generate
	@go tool sqlc generate -f sqlc/sqlc.yml
	@./vendor-bin/tailwindcss -i tailwind/styles.css -o static/public/styles-tw.css --minify
	@echo "Code generation complete."

generate-watch:
	@echo "Generating code (skipping templ, handled by watcher)..."
	@go generate ./...
	@go tool sqlc generate -f sqlc/sqlc.yml
	@./vendor-bin/tailwindcss -i tailwind/styles.css -o static/public/styles-tw.css --minify
	@echo "Code generation complete."

run:
	@echo "Running the application..."
	@go run cmd/*.go -addr ":$(PORT)" -watch -dburl ":memory:"

build:
	@go build ./...

test:
	@go test ./...

format:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

bench:
	@go test -bench=. -benchmem ./...

docker-build:
	@echo "Building Docker image..."
	@$(CONTAINER_RUNTIME) build --platform linux/amd64 -t $(IMAGE) .
	@$(foreach tag,$(EXTRA_TAGS),$(CONTAINER_RUNTIME) tag $(IMAGE) $(REGISTRY):$(tag);)

docker-push:
	@echo "Pushing Docker image..."
	@$(CONTAINER_RUNTIME) push $(IMAGE)
	@$(foreach tag,$(EXTRA_TAGS),$(CONTAINER_RUNTIME) push $(REGISTRY):$(tag);)

docker-build-push: docker-build docker-push
	@echo "Docker image built and pushed successfully."

ci-pr: build format test bench docker-build

ci-main: build format test bench docker-build docker-push

ci-release: build format test bench docker-build docker-push