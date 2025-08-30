
watch-tw:
	@echo "Watching for changes..."
	@./vendor-bin/tailwindcss -i static/tailwind/styles.css -o static/public/styles-tw.css --watch

watch-templ:
	@go tool templ generate --watch --proxy="http://localhost:3006" --cmd="go run cmd/main.go -addr :3006 -watch -dburl tmp.db.sqlite"

generate:
	@echo "Generating code..."
	@go generate ./...
	@go tool templ generate
	@go tool sqlc generate -f sqlc/sqlc.yml
	@echo "Code generation complete."

run:
	@echo "Running the application..."
	@go run cmd/*.go -addr ":3006" -watch -dburl ":memory:"
