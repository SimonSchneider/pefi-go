PORT := 3002
IMAGE ?= ghcr.io/simonschneider/pefigo:main

watch-tw:
	@echo "Watching for changes..."
	@./vendor-bin/tailwindcss -i tailwind/styles.css -o static/public/styles-tw.css --watch

watch-templ:
	@go tool templ generate --watch --proxy="http://localhost:$(PORT)" --cmd="go run cmd/main.go -addr :$(PORT) -watch -dburl tmp.db.sqlite"

generate:
	@echo "Generating code..."
	@go generate ./...
	@go tool templ generate
	@go tool sqlc generate -f sqlc/sqlc.yml
	@./vendor-bin/tailwindcss -i tailwind/styles.css -o static/public/styles-tw.css --minify
	@echo "Code generation complete."

run:
	@echo "Running the application..."
	@go run cmd/*.go -addr ":$(PORT)" -watch -dburl ":memory:"

docker-build:
	@echo "Building Docker image..."
	@podman build --platform linux/amd64 -t $(IMAGE) .

docker-push:
	@echo "Pushing Docker image..."
	@podman push $(IMAGE)

docker-build-push: docker-build docker-push
	@echo "Docker image built and pushed successfully."