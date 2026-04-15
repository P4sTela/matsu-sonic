.PHONY: dev build clean test

# Development: frontend HMR + Go server
dev:
	cd frontend && npm run dev &
	go run . -port 8765

# Production build: frontend → embed → Go binary
build:
	cd frontend && npm ci && npm run build
	CGO_ENABLED=1 go build -o gdrive-sync -ldflags="-s -w" .

# Cross compile for Linux
build-linux:
	cd frontend && npm ci && npm run build
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o gdrive-sync-linux -ldflags="-s -w" .

# Cross compile for Windows
build-windows:
	cd frontend && npm ci && npm run build
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o gdrive-sync.exe -ldflags="-s -w" .

# Run all tests
test:
	go test ./...

clean:
	rm -f gdrive-sync gdrive-sync-linux gdrive-sync.exe
	rm -rf frontend/dist
