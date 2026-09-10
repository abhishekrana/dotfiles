//! The binary's contract: exit codes, stdin, output modes, and the messages a caller sees.
#![allow(clippy::expect_used)]

use std::io::Write as _;
use std::process::{Command, Output, Stdio};

fn folio(args: &[&str], stdin: Option<&str>) -> Output {
    let mut cmd = Command::new(env!("CARGO_BIN_EXE_folio"));
    cmd.args(args)
        .env("DOTFILES_TRACE", "0")
        .stdin(if stdin.is_some() { Stdio::piped() } else { Stdio::null() });
    cmd.stdout(Stdio::piped()).stderr(Stdio::piped());
    let mut child = cmd.spawn().expect("folio starts");
    if let (Some(text), Some(mut pipe)) = (stdin, child.stdin.take()) {
        pipe.write_all(text.as_bytes()).expect("write stdin");
    }
    child.wait_with_output().expect("folio exits")
}

fn stderr(out: &Output) -> String {
    String::from_utf8_lossy(&out.stderr).into_owned()
}

#[test]
fn a_missing_file_exits_one_with_the_path() {
    let out = folio(&["--inline", "/no/such/note.md"], None);
    assert_eq!(out.status.code(), Some(1));
    assert!(stderr(&out).contains("/no/such/note.md"), "{}", stderr(&out));
}

#[test]
fn an_unknown_style_exits_two_and_lists_the_builtins() {
    let out = folio(&["--inline", "--style", "nope", "tests/fixtures/sample.md"], None);
    assert_eq!(out.status.code(), Some(2));
    assert!(stderr(&out).contains("github"), "{}", stderr(&out));
}

#[test]
fn an_unknown_theme_exits_one_and_lists_the_flavors() {
    let out = folio(&["--inline", "--theme", "mocha", "tests/fixtures/sample.md"], None);
    assert_eq!(out.status.code(), Some(1));
    assert!(stderr(&out).contains("catppuccin-mocha"), "{}", stderr(&out));
}

#[test]
fn stdin_renders_plain_in_a_pipe_and_ansi_when_asked() {
    let plain = folio(&["--inline", "--width", "40"], Some("# Hi\n\nSome *text*.\n"));
    assert!(plain.status.success());
    let text = String::from_utf8_lossy(&plain.stdout);
    assert!(text.contains("Hi") && !text.contains('\u{1b}'), "{text}");
    let ansi = folio(&["--inline", "--format", "ansi", "--width", "40"], Some("# Hi\n"));
    assert!(
        String::from_utf8_lossy(&ansi.stdout).contains("\u{1b}[1;38;2;"),
        "bold coloured heading"
    );
}

#[test]
fn format_requires_inline_and_a_terminal_less_run_without_a_file_fails() {
    let out = folio(&["--format", "plain", "tests/fixtures/sample.md"], None);
    assert_eq!(out.status.code(), Some(2), "clap usage error");
    let out = folio(&[], None);
    assert_eq!(out.status.code(), Some(1));
    assert!(stderr(&out).contains("not a terminal"), "{}", stderr(&out));
    let out = folio(&["--inline"], None);
    assert!(out.status.success(), "an empty pipe is an empty document, not an error");
}

#[test]
fn list_styles_and_help_are_available() {
    let out = folio(&["--list-styles"], None);
    let text = String::from_utf8_lossy(&out.stdout);
    assert!(out.status.success() && text.contains("github") && text.contains("base"));
    let help = folio(&["--help"], None);
    let text = String::from_utf8_lossy(&help.stdout);
    assert!(
        text.contains("--inline") && text.contains("--no-watch") && text.contains("FOLIO_THEME"),
        "{text}"
    );
}
