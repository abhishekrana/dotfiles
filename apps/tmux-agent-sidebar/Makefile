.PHONY: build test unit e2e install-hooks clean

build:
	go build -o bin/tmux-agent-sidebar ./cmd/tmux-agent-sidebar

test:
	go test ./...

unit:
	go test -short ./...

# Full lifecycle against throwaway tmux servers (private sockets; never
# touches your live tmux). Needs tmux installed.
e2e:
	go test ./e2e/ -v -count=1

install-hooks: build
	./bin/tmux-agent-sidebar install-hooks

clean:
	rm -rf bin
