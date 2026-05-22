# Build the Go server
build:
    go build ./cmd/sour

# Install client dependencies
install:
    cd client && bun install

# Build the client for production
build-client:
    cd client && bun run build

# Build everything
build-all: build-client build

# Start the Vite dev server with HMR
dev-client:
    cd client && bun run dev

# Start the Go server with hot reload
dev-server:
    go tool wgo run ./cmd/sour serve test-config.yaml

# Start everything for development
dev:
    ./scripts/dev

# Run Go tests
test:
    go test ./...

# Type-check client
check-client:
    cd client && bunx tsc --noEmit
