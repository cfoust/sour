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
    #!/usr/bin/env bash
    set -e
    go tool wgo run ./cmd/sour serve test-config.yaml &
    SERVER_PID=$!

    # Wait for Go server to be ready
    echo "Waiting for Go server on :1337..."
    for i in $(seq 1 30); do
      if curl -s -o /dev/null http://localhost:1337/api/client-config.js 2>/dev/null; then
        break
      fi
      sleep 1
    done

    cd client && bun run dev &
    CLIENT_PID=$!
    trap "kill $CLIENT_PID $SERVER_PID 2>/dev/null" EXIT
    echo ""
    echo "  Client (Vite HMR): http://localhost:5173"
    echo "  Go server:         http://localhost:1337"
    echo ""
    wait

# Run Go tests
test:
    go test ./...

# Type-check client
check-client:
    cd client && bunx tsc --noEmit

# Derive game modes from a map file
modes file:
    go run ./cmd/sourdump modes {{file}}
