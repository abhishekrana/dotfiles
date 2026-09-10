//! Hostile and oversized input: nothing panics, no row exceeds the pane, and layout stays fast.
#![allow(clippy::expect_used)]

use std::time::{Duration, Instant};

use folio::app::{App, Msg};
use folio::buffer::Buffer;
use folio::layout::{Layouter, Page};
use folio::theme::Theme;
use folio::{doc, render, style};
use unicode_width::UnicodeWidthStr;

const SAMPLE: &str = include_str!("fixtures/sample.md");

/// Concatenates `f(i)` for every `i`, the plain way clippy asks for.
fn build(n: usize, f: impl Fn(usize) -> String) -> String {
    let mut out = String::new();
    for i in 0..n {
        out.push_str(&f(i));
    }
    out
}

/// Generous for a debug build on a busy CI runner; the release binary does this in well under 200 ms.
const LAYOUT_BUDGET: Duration = Duration::from_secs(5);

fn page(text: &str, width: u16) -> Page {
    let document = doc::parse(&Buffer::from_text(text));
    let style = style::load("github", None).expect("style");
    Layouter::new(Theme::default_theme().expect("theme")).layout(&document, &style, width)
}

fn assert_fits(text: &str, width: u16) {
    let p = page(text, width);
    let widest = render::plain(&p).lines().map(UnicodeWidthStr::width).max().unwrap_or(0);
    assert!(
        widest <= usize::from(width.max(1)),
        "width {width}: a row is {widest} cells"
    );
}

#[test]
fn a_ten_thousand_line_document_lays_out_within_budget() {
    let big = build(220, |i| {
        SAMPLE.replace("# Backing up", &format!("# Section {i} backing up"))
    });
    assert!(big.lines().count() > 10_000);
    let started = Instant::now();
    let p = page(&big, 100);
    let took = started.elapsed();
    assert!(p.lines.len() > 10_000);
    assert!(took < LAYOUT_BUDGET, "parse and layout took {took:?}");
}

#[test]
fn deep_nesting_does_not_overflow() {
    let quotes = build(400, |i| format!("{} q\n", ">".repeat(i)));
    assert_fits(&quotes, 100);
    let lists = build(300, |i| format!("{}- item\n", "  ".repeat(i)));
    assert_fits(&lists, 100);
}

#[test]
fn oversized_lines_and_tables_are_clipped_to_the_pane() {
    assert_fits(&format!("{}\n", "x".repeat(200_000)), 80);
    let header = build(60, |i| format!("| c{i} "));
    let rule = "|---".repeat(60);
    let row = build(60, |_| format!("| {} ", "x".repeat(20)));
    let table = format!("{header}|\n{rule}|\n{row}|\n");
    for width in [12u16, 40, 100, 400] {
        assert_fits(&table, width);
    }
}

#[test]
fn malformed_and_odd_markdown_renders_something() {
    let odd = concat!(
        "# H\n\n\u{200b}text\twith\u{1b}[31mansi\u{1b}[0m and RTL: \u{5e9}\u{5dc}\u{5d5}\u{5dd} ",
        "and combining: e\u{301}\n\n|a|\n|-|\n\n| no | header rule\n| a | b | c | d |\n\n```\nunterminated fence\n"
    );
    assert_fits(odd, 40);
    assert!(!render::plain(&page(odd, 40)).trim().is_empty());
    for text in [
        "",
        "\n\n\n",
        "---\ntype: x\n---\n",
        "# only a heading",
        "```\n```",
        "| |\n|-|\n| |",
    ] {
        for width in [1u16, 2, 5, 10, 24, 1000] {
            assert_fits(text, width);
        }
    }
}

#[test]
fn the_reader_survives_a_tiny_or_empty_terminal() {
    let style = style::load("github", None).expect("style");
    let theme = Theme::default_theme().expect("theme");
    let mut app = App::new(Buffer::from_text(SAMPLE), style, theme);
    for (w, h) in [(0u16, 0u16), (1, 1), (5, 2), (1000, 1), (100, 1000)] {
        app.resize(w, h);
        for msg in [
            Msg::HalfPage(1),
            Msg::Bottom,
            Msg::NextHeading,
            Msg::ToggleOutline,
            Msg::Down,
            Msg::Select,
            Msg::Top,
        ] {
            app.update(msg);
        }
        app.update(Msg::Click {
            col: u16::MAX,
            row: u16::MAX,
        });
        app.update(Msg::Click { col: 0, row: 0 });
        assert!(app.scroll() <= app.max_scroll());
    }
}

#[test]
fn invalid_utf8_is_an_error_not_a_panic() {
    let bytes: Vec<u8> = (0..=255u8).collect();
    let err = Buffer::from_reader(bytes.as_slice()).expect_err("invalid UTF-8 is refused");
    assert!(err.to_string().starts_with("cannot read stdin"));
}
