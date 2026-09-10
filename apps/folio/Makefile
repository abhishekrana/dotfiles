.PHONY: build test lint clean

# Release binary into bin/, the uniform apps/ contract bootstrap.sh builds and links.
# Copy then rename, so a folio that is running keeps its old file and the link picks up the new one.
build:
	cargo build --release --quiet
	mkdir -p bin
	cp target/release/folio bin/folio.tmp
	mv -f bin/folio.tmp bin/folio

# Tests never write to the shared trace log.
test:
	DOTFILES_TRACE=0 cargo test --quiet

# Formatting and clippy at warnings-as-errors; `task check` runs this.
lint:
	cargo fmt --check
	cargo clippy --all-targets --quiet -- -D warnings

clean:
	cargo clean
	rm -rf bin
