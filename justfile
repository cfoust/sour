# Build the Go server
build:
    go build ./cmd/sour

# Build the client (production)
build-client:
    cd client && yarn build

# Start both dev servers (client watcher + sour server)
dev: build
    #!/usr/bin/env bash
    set -e

    # Start client watcher in background
    cd client && yarn serve &
    CLIENT_PID=$!

    # Wait for initial build
    sleep 3

    # Start sour server in dev mode
    cd "{{justfile_directory()}}" && ./sour serve --dev test-config.yaml &
    SERVER_PID=$!

    trap "kill $CLIENT_PID $SERVER_PID 2>/dev/null" EXIT
    echo "Dev servers running — http://localhost:1337"
    echo "  Client watcher PID: $CLIENT_PID"
    echo "  Sour server PID:    $SERVER_PID"
    echo "Press Ctrl+C to stop."
    wait

# Start just the client watcher
dev-client:
    cd client && yarn serve

# Start just the sour server in dev mode
dev-server: build
    ./sour serve --dev test-config.yaml

# Run Go tests
test:
    go test ./...

# Build sourdump
build-sourdump:
    go build ./cmd/sourdump

# Derive game modes from a map file
modes file:
    ./sourdump modes {{file}}
