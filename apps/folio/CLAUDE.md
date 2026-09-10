# CLAUDE.md

folio: a markdown reader for the terminal that reads like a page. Rust, ratatui, comrak. `DESIGN.md` is the spec;
read it before changing what anything looks like. Line width ≤120 everywhere (`rustfmt.toml`).

## Commands

From the repo root, `task folio:build` / `folio:lint` / `folio:test` delegate to these; `task check` runs lint and test.

```bash
make build                    # release binary at bin/folio (bootstrap links it into ~/.local/bin)
make lint                     # cargo fmt --check + clippy --all-targets -D warnings
make test                     # unit tests and insta snapshots
INSTA_UPDATE=always cargo test  # accept changed snapshots, then read the diff before committing
bin/folio tests/fixtures/sample.md            # the reader
bin/folio --inline tests/fixtures/sample.md   # to stdout: ANSI on a terminal, plain in a pipe
FOLIO_LOG=debug bin/folio FILE                # log to ~/.local/state/folio/folio.log
```

## Two rules

- **Styles are files.** A look is `styles/<name>.toml`: one table per element, closed field sets, `extends` for
  composition. Nothing in `src/` knows what GitHub looks like. A new look is a new file; a rendering change that is
  not expressible in a style file is a schema change in `src/style/mod.rs` and `styles/base.toml` together.
- **Colours are roles.** Styles name `accent`, `surface`, `muted`; `src/theme` maps roles to hexes from
  `design/palette.toml`, which is `include_str!`'d so a flavor is defined once. Never a hex in `src/` or a style.

## Layout of the code

`buffer` (rope) -> `doc` (blocks and inlines, every node with a byte `Span`) -> `layout` (rows of `Segment`s at a
measure, cached per block and width) -> `render` (ratatui or ANSI/plain) with `app` running the Elm loop and `ui`
drawing. `main.rs` is the CLI edge and the only place `anyhow` appears.

## Conventions

- Pinned everything: `=x.y.z` in `Cargo.toml`, `Cargo.lock` committed, `rust-toolchain.toml` agrees with
  `RUST_VERSION` in `install.sh` (`task conf` checks). Bumps are their own commits.
- `[lints]` in `Cargo.toml`: pedantic clippy, no `unwrap`/`expect` outside tests (`clippy.toml`), no `unsafe`.
- Errors are `thiserror` enums per module that name the file and key; `main` maps a `StyleError` to exit 2, anything
  else to 1.
- Tests: unit tests next to the code, snapshots in `tests/snapshots/` from `tests/fixtures/sample.md`. A snapshot diff
  is a reviewed rendering change, never a rubber stamp.
- Comments state what the code does now, one line. History belongs in the commit message.
