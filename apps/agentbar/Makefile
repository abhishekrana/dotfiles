.PHONY: gen build test unit vet e2e clean

# A test must never write to the developer's state dir: the trace log and the
# pin mirror both live under XDG_STATE_HOME, and a fixture session name that
# happened to match a real one would otherwise pin it in the live sidebar.
# Redirected, not disabled - two e2e tests assert on what the sidebar traced.
# The e2e harness also redirects per test; this covers the unit tests.
TEST_ENV = XDG_STATE_HOME=$(CURDIR)/bin/test-state

# theme_gen.go is generated from design/palette.toml so a flavor is defined once.
# The output is committed, so a plain `go build` still works without this.
gen:
	@./scripts/gen-theme.sh

# Two binaries from one module: agentbar is the sidebar, workdesk is the GitLab work
# inbox. Separate commands so a forge failure is never a sidebar failure, sharing
# internal/ so there is one trace writer and one tmux reader.
build: gen
	go build -o bin/agentbar ./cmd/agentbar
	go build -o bin/workdesk ./cmd/workdesk

test: gen vet
	$(TEST_ENV) go test ./...

unit:
	$(TEST_ENV) go test -short ./...

# gofmt and vet are not run anywhere else in this repo, so they run here. `gofmt -l`
# exits 0 even when it lists files, so the listing itself has to be the failure.
vet:
	@unformatted=$$(gofmt -l cmd internal e2e); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...

# Full lifecycle against throwaway tmux servers (private sockets; never
# touches your live tmux). Needs tmux installed.
e2e:
	$(TEST_ENV) go test ./e2e/ -v -count=1

clean:
	rm -rf bin
