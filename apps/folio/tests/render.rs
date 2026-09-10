//! Snapshot tests: the sample note through each built-in style at two widths.
#![allow(clippy::expect_used)]

use folio::buffer::Buffer;
use folio::layout::Layouter;
use folio::theme::Theme;
use folio::{doc, render, style};

const SAMPLE: &str = include_str!("fixtures/sample.md");

fn render_plain(style_name: &str, width: u16) -> String {
    let buffer = Buffer::from_text(SAMPLE);
    let document = doc::parse(&buffer);
    let style = style::load(style_name, None).expect("built-in style loads");
    let page = Layouter::new().layout(&document, &style, width);
    render::plain(&page)
}

#[test]
fn github_at_100_columns() {
    insta::assert_snapshot!(render_plain("github", 100));
}

#[test]
fn github_at_64_columns() {
    insta::assert_snapshot!(render_plain("github", 64));
}

#[test]
fn github_ansi_carries_theme_colours_and_links() {
    let buffer = Buffer::from_text(SAMPLE);
    let document = doc::parse(&buffer);
    let style = style::load("github", None).expect("style");
    let page = Layouter::new().layout(&document, &style, 100);
    let theme = Theme::by_id("solarized-light").expect("theme");
    let out = render::ansi(&page, theme, true);
    assert!(out.contains("38;2;38;139;210"), "accent blue is painted");
    assert!(!out.contains("\x1b]8;;wiki:"), "wikilinks are not OSC 8 hyperlinks");
    insta::assert_snapshot!(out);
}
