//! Snapshot tests: the sample note through each built-in style at two widths.
#![allow(clippy::expect_used)]

use folio::buffer::Buffer;
use folio::layout::Layouter;
use folio::theme::Theme;
use folio::{doc, render, style};

const SAMPLE: &str = include_str!("fixtures/sample.md");
const EDGES: &str = include_str!("fixtures/edges.md");

fn render_plain(style_name: &str, width: u16) -> String {
    render_text(SAMPLE, style_name, width)
}

fn render_text(text: &str, style_name: &str, width: u16) -> String {
    let buffer = Buffer::from_text(text);
    let document = doc::parse(&buffer);
    let style = style::load(style_name, None).expect("built-in style loads");
    let theme = Theme::default_theme().expect("palette default");
    let page = Layouter::new(theme).layout(&document, &style, width);
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
    let theme = Theme::by_id("solarized-light").expect("theme");
    let page = Layouter::new(theme).layout(&document, &style, 100);
    let out = render::ansi(&page, theme, true);
    assert!(out.contains("38;2;38;139;210"), "accent blue is painted");
    assert!(!out.contains("\x1b]8;;wiki:"), "wikilinks are not OSC 8 hyperlinks");
    insta::assert_snapshot!(out);
}

#[test]
fn every_flavor_paints_the_sample() {
    let buffer = Buffer::from_text(SAMPLE);
    let document = doc::parse(&buffer);
    let style = style::load("github", None).expect("style");
    for theme in folio::theme::flavors().expect("palette") {
        let page = Layouter::new(theme).layout(&document, &style, 100);
        insta::assert_snapshot!(format!("ansi_{}", theme.id), render::ansi(&page, theme, false));
    }
}

#[test]
fn edge_cases_at_100_columns() {
    insta::assert_snapshot!(render_text(EDGES, "github", 100));
}

#[test]
fn edge_cases_never_exceed_the_pane() {
    for width in [24u16, 40, 64, 100] {
        let out = render_text(EDGES, "github", width);
        let widest = out
            .lines()
            .map(unicode_width::UnicodeWidthStr::width)
            .max()
            .unwrap_or(0);
        assert!(widest <= usize::from(width), "width {width}: a row is {widest} cells");
    }
}
