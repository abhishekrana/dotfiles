.PHONY: build test unit e2e clean

# A test must never write to the developer's state dir: the trace log and the
# pin mirror both live under XDG_STATE_HOME, and a fixture session name that
# happened to match a real one would otherwise pin it in the live sidebar.
# Redirected, not disabled - two e2e tests assert on what the sidebar traced.
# The e2e harness also redirects per test; this covers the unit tests.
TEST_ENV = XDG_STATE_HOME=$(CURDIR)/bin/test-state

build:
	go build -o bin/agentbar ./cmd/agentbar

test:
	$(TEST_ENV) go test ./...

unit:
	$(TEST_ENV) go test -short ./...

# Full lifecycle against throwaway tmux servers (private sockets; never
# touches your live tmux). Needs tmux installed.
e2e:
	$(TEST_ENV) go test ./e2e/ -v -count=1

clean:
	rm -rf bin
