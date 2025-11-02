build:
    go build -o bin/sessionwardend ./cmd/sessionwardend
    go build -o bin/swctl ./cmd/swctl

tidy:
    go mod tidy

test:
    go test ./...

run EXE:
    go build -o bin/{{EXE}} ./cmd/{{EXE}}
    ./bin/{{EXE}}
