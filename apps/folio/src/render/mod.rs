//! Painting: the same lines to ratatui for the TUI or to text for `--inline`.

use std::fmt::Write as _;

use ratatui::style::{Color, Modifier, Style as RStyle};
use ratatui::text::{Line as RLine, Span as RSpan};

use crate::layout::{CellStyle, Line, Page};
use crate::theme::{Rgb, Theme};

/// The page as plain text, one row per line, margins included.
#[must_use]
pub fn plain(page: &Page) -> String {
    let mut out = String::new();
    for line in &page.lines {
        push_margin(&mut out, page.left);
        for s in &line.segments {
            out.push_str(&s.text);
        }
        trim_line_end(&mut out);
        out.push('\n');
    }
    out
}

/// The page with SGR colours from the theme; `hyperlinks` wraps links in OSC 8.
#[must_use]
pub fn ansi(page: &Page, theme: &Theme, hyperlinks: bool) -> String {
    let mut out = String::new();
    for line in &page.lines {
        push_margin(&mut out, page.left);
        for s in &line.segments {
            let sgr = sgr(s.style, theme);
            let link = s.link.as_deref().filter(|l| hyperlinks && !l.starts_with("wiki:"));
            if let Some(url) = link {
                let _ = write!(out, "\x1b]8;;{url}\x1b\\");
            }
            if sgr.is_empty() {
                out.push_str(&s.text);
            } else {
                let _ = write!(out, "\x1b[{sgr}m{}\x1b[0m", s.text);
            }
            if link.is_some() {
                out.push_str("\x1b]8;;\x1b\\");
            }
        }
        out.push('\n');
    }
    out
}

fn sgr(cs: CellStyle, theme: &Theme) -> String {
    let mut codes: Vec<String> = Vec::new();
    if cs.bold {
        codes.push("1".into());
    }
    if cs.italic {
        codes.push("3".into());
    }
    if cs.underline {
        codes.push("4".into());
    }
    if cs.strike {
        codes.push("9".into());
    }
    if let Some(role) = cs.fg {
        let Rgb(r, g, b) = theme.color(role);
        codes.push(format!("38;2;{r};{g};{b}"));
    }
    if let Some(role) = cs.bg {
        let Rgb(r, g, b) = theme.color(role);
        codes.push(format!("48;2;{r};{g};{b}"));
    }
    codes.join(";")
}

fn push_margin(out: &mut String, left: u16) {
    for _ in 0..left {
        out.push(' ');
    }
}

fn trim_line_end(out: &mut String) {
    let len = out.trim_end_matches(' ').len();
    out.truncate(len);
}

/// One page row as a ratatui line, margin included.
#[must_use]
pub fn to_ratatui(line: &Line, theme: &Theme, left: u16) -> RLine<'static> {
    let mut spans = Vec::with_capacity(line.segments.len() + 1);
    if left > 0 {
        spans.push(RSpan::raw(" ".repeat(usize::from(left))));
    }
    for s in &line.segments {
        spans.push(RSpan::styled(s.text.clone(), ratatui_style(s.style, theme)));
    }
    RLine::from(spans)
}

/// A theme colour as a ratatui colour.
#[must_use]
pub fn color(rgb: Rgb) -> Color {
    Color::Rgb(rgb.0, rgb.1, rgb.2)
}

fn ratatui_style(cs: CellStyle, theme: &Theme) -> RStyle {
    let mut st = RStyle::default();
    if let Some(role) = cs.fg {
        st = st.fg(color(theme.color(role)));
    }
    if let Some(role) = cs.bg {
        st = st.bg(color(theme.color(role)));
    }
    let mut m = Modifier::empty();
    m.set(Modifier::BOLD, cs.bold);
    m.set(Modifier::ITALIC, cs.italic);
    m.set(Modifier::UNDERLINED, cs.underline);
    m.set(Modifier::CROSSED_OUT, cs.strike);
    st.add_modifier(m)
}
