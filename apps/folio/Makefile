.PHONY: build test lint clean

# Release binary into bin/, the uniform apps/ contract bootstrap.sh builds and links.
build:
	cargo build --release --quiet
	mkdir -p bin
	cp target/release/folio bin/folio

test:
	cargo test --quiet

# Formatting and clippy at warnings-as-errors; `task check` runs this.
lint:
	cargo fmt --check
	cargo clippy --all-targets --quiet -- -D warnings

clean:
	cargo clean
	rm -rf bin
