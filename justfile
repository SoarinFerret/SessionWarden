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

vm:
    rm nixos.qcow2 || true
    nix build .#nixosConfigurations.x86_64-linux.vm.config.system.build.vm
    ./result/bin/run-nixos-vm
