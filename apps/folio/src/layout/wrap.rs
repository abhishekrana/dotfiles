//! Inline runs -> words -> wrapped rows of segments.

use std::sync::Arc;

use unicode_segmentation::UnicodeSegmentation;
use unicode_width::UnicodeWidthStr;

use super::{CellStyle, Segment};
use crate::doc::{Inline, Span};
use crate::style::{InlineRule, Style};

const SUPERSCRIPT: [char; 10] = ['⁰', '¹', '²', '³', '⁴', '⁵', '⁶', '⁷', '⁸', '⁹'];

/// A word, a space, or a forced break, ready for greedy wrapping.
#[derive(Debug, Clone)]
pub(super) struct Word {
    text: String,
    style: CellStyle,
    src: Option<Span>,
    link: Option<Arc<str>>,
    width: usize,
    kind: Kind,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Kind {
    Text,
    Space,
    Break,
}

/// Flattens inlines into words, applying each element's rule on top of `base`.
pub(super) fn words(inlines: &[Inline], base: CellStyle, style: &Style) -> Vec<Word> {
    let mut out = Vec::new();
    flatten(inlines, base, None, style, &mut out);
    out
}

fn flatten(inlines: &[Inline], base: CellStyle, link: Option<&Arc<str>>, style: &Style, out: &mut Vec<Word>) {
    for inline in inlines {
        match inline {
            Inline::Text(text, span) => push_text(text, *span, base, link, out),
            Inline::Code(code, span) => {
                let rule = &style.code_span;
                let pad = " ".repeat(usize::from(rule.pad));
                let cs = CellStyle {
                    fg: Some(rule.fg),
                    bg: rule.bg,
                    ..base
                };
                out.push(word(format!("{pad}{code}{pad}"), cs, Some(*span), link.cloned()));
            }
            Inline::Strong(inner, _) => flatten(inner, apply(base, &style.strong), link, style, out),
            Inline::Emph(inner, _) => flatten(inner, apply(base, &style.emph), link, style, out),
            Inline::Strike(inner, _) => flatten(inner, apply(base, &style.strike), link, style, out),
            Inline::Link { text, href, .. } => {
                let href: Arc<str> = Arc::from(href.as_str());
                flatten(text, apply(base, &style.link), Some(&href), style, out);
            }
            Inline::WikiLink { target, alias, span } => {
                let shown = alias.as_deref().unwrap_or(target);
                let target: Arc<str> = Arc::from(format!("wiki:{target}"));
                push_text(shown, *span, apply(base, &style.wikilink), Some(&target), out);
            }
            Inline::Tag(name, span) => {
                out.push(word(format!("#{name}"), apply(base, &style.tag), Some(*span), None));
            }
            Inline::FootnoteRef(label, span) => {
                let cs = CellStyle {
                    fg: Some(style.footnote.fg),
                    ..base
                };
                let mut w = word(footnote_marker(label, style), cs, Some(*span), None);
                // A footnote mark clings to the word before it.
                if let Some(prev) = out.last_mut().filter(|p| p.kind == Kind::Text) {
                    prev.text.push_str(&w.text);
                    prev.width += w.width;
                    continue;
                }
                w.kind = Kind::Text;
                out.push(w);
            }
            Inline::Image { alt, span, .. } => {
                let cs = CellStyle {
                    fg: Some(style.image.fg),
                    ..base
                };
                push_text(&format!("{} {alt}", style.image.placeholder), *span, cs, None, out);
            }
            Inline::SoftBreak => out.push(space(base)),
            Inline::HardBreak => out.push(Word {
                kind: Kind::Break,
                ..space(base)
            }),
        }
    }
}

pub(super) fn footnote_marker(label: &str, style: &Style) -> String {
    match style.footnote.marker {
        crate::style::FootnoteMarker::Superscript if label.bytes().all(|b| b.is_ascii_digit()) => {
            label.bytes().map(|b| SUPERSCRIPT[usize::from(b - b'0')]).collect()
        }
        _ => format!("[{label}]"),
    }
}

fn apply(base: CellStyle, rule: &InlineRule) -> CellStyle {
    CellStyle {
        fg: rule.fg.or(base.fg),
        fg_rgb: base.fg_rgb,
        bg: rule.bg.or(base.bg),
        bold: base.bold || rule.bold,
        italic: base.italic || rule.italic,
        underline: base.underline || rule.underline,
        strike: base.strike || rule.strike,
    }
}

fn push_text(text: &str, span: Span, style: CellStyle, link: Option<&Arc<str>>, out: &mut Vec<Word>) {
    let mut offset = 0;
    for piece in text.split_inclusive(char::is_whitespace) {
        let trimmed = piece.trim_end_matches(char::is_whitespace);
        if !trimmed.is_empty() {
            let src = Span {
                start: span.start + offset,
                end: span.start + offset + trimmed.len(),
            };
            out.push(word(trimmed.to_owned(), style, Some(src), link.cloned()));
        }
        if trimmed.len() < piece.len() {
            let src = Span {
                start: span.start + offset + trimmed.len(),
                end: span.start + offset + piece.len(),
            };
            out.push(Word {
                src: Some(src),
                link: link.cloned(),
                ..space(style)
            });
        }
        offset += piece.len();
    }
}

#[cfg(test)]
impl Word {
    pub(super) fn plain(text: &str, style: CellStyle) -> Self {
        word(text.to_owned(), style, None, None)
    }

    pub(super) fn gap(style: CellStyle) -> Self {
        space(style)
    }
}

fn word(text: String, style: CellStyle, src: Option<Span>, link: Option<Arc<str>>) -> Word {
    Word {
        width: text.width(),
        text,
        style,
        src,
        link,
        kind: Kind::Text,
    }
}

fn space(style: CellStyle) -> Word {
    Word {
        text: " ".to_owned(),
        style,
        src: None,
        link: None,
        width: 1,
        kind: Kind::Space,
    }
}

/// Greedy wrap at `width` cells. A word wider than the row is split on grapheme boundaries.
pub(super) fn wrap(words: &[Word], width: usize) -> Vec<Vec<Segment>> {
    let width = width.max(1);
    let mut rows: Vec<Vec<Segment>> = vec![Vec::new()];
    let mut used = 0usize;
    for w in words {
        match w.kind {
            Kind::Break => {
                rows.push(Vec::new());
                used = 0;
            }
            Kind::Space => {
                if used > 0 && used < width {
                    push_seg(rows.last_mut(), w);
                    used += 1;
                }
            }
            Kind::Text if w.width <= width => {
                if used + w.width > width {
                    trim_trailing_space(rows.last_mut());
                    rows.push(Vec::new());
                    used = 0;
                }
                push_seg(rows.last_mut(), w);
                used += w.width;
            }
            Kind::Text => {
                for piece in split_graphemes(w, width) {
                    if used + piece.width > width {
                        trim_trailing_space(rows.last_mut());
                        rows.push(Vec::new());
                        used = 0;
                    }
                    push_seg(rows.last_mut(), &piece);
                    used += piece.width;
                }
            }
        }
    }
    for row in &mut rows {
        trim_trailing_space(Some(row));
    }
    if rows.len() > 1 && rows.last().is_some_and(Vec::is_empty) {
        rows.pop();
    }
    rows
}

fn split_graphemes(w: &Word, width: usize) -> Vec<Word> {
    let mut pieces = Vec::new();
    let mut cur = String::new();
    let mut cur_w = 0;
    for g in w.text.graphemes(true) {
        let gw = g.width();
        if cur_w + gw > width && !cur.is_empty() {
            pieces.push(Word {
                text: std::mem::take(&mut cur),
                width: cur_w,
                ..w.clone()
            });
            cur_w = 0;
        }
        cur.push_str(g);
        cur_w += gw;
    }
    if !cur.is_empty() {
        pieces.push(Word {
            text: cur,
            width: cur_w,
            ..w.clone()
        });
    }
    pieces
}

/// Appends a word, merging into the previous segment when style, link and source line up.
fn push_seg(row: Option<&mut Vec<Segment>>, w: &Word) {
    let Some(row) = row else { return };
    if let Some(last) = row.last_mut()
        && last.style == w.style
        && last.link == w.link
        && contiguous(last.src, w.src)
    {
        last.text.push_str(&w.text);
        if let (Some(a), Some(b)) = (last.src.as_mut(), w.src) {
            a.end = b.end;
        }
        return;
    }
    row.push(Segment {
        text: w.text.clone(),
        style: w.style,
        src: w.src,
        link: w.link.clone(),
    });
}

fn contiguous(a: Option<Span>, b: Option<Span>) -> bool {
    match (a, b) {
        (None, None) => true,
        (Some(a), Some(b)) => a.end == b.start,
        (Some(_), None) | (None, Some(_)) => false,
    }
}

fn trim_trailing_space(row: Option<&mut Vec<Segment>>) {
    if let Some(row) = row
        && let Some(last) = row.last_mut()
    {
        let trimmed = last.text.trim_end_matches(' ').len();
        if trimmed == 0 {
            row.pop();
        } else {
            last.text.truncate(trimmed);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::doc::Inline;

    fn style() -> Style {
        crate::style::load("github", None).expect("style")
    }

    fn text(s: &str) -> Vec<Inline> {
        vec![Inline::Text(s.to_owned(), Span { start: 0, end: s.len() })]
    }

    fn rows(inlines: &[Inline], width: usize) -> Vec<String> {
        wrap(&words(inlines, CellStyle::default(), &style()), width)
            .iter()
            .map(|r| r.iter().map(|s| s.text.as_str()).collect())
            .collect()
    }

    #[test]
    fn wraps_at_spaces_and_drops_edge_spaces() {
        assert_eq!(
            rows(&text("the quick brown fox jumps"), 10),
            vec!["the quick", "brown fox", "jumps"]
        );
    }

    #[test]
    fn a_long_word_is_split_by_grapheme() {
        assert_eq!(rows(&text("abcdefghij"), 4), vec!["abcd", "efgh", "ij"]);
    }

    #[test]
    fn wide_characters_count_two_cells() {
        assert_eq!(rows(&text("日本語 テキスト"), 6), vec!["日本語", "テキス", "ト"]);
    }

    #[test]
    fn merged_segments_keep_contiguous_spans() {
        let r = wrap(&words(&text("ab cd"), CellStyle::default(), &style()), 80);
        assert_eq!(r[0].len(), 1);
        assert_eq!(r[0][0].src, Some(Span { start: 0, end: 5 }));
    }
}
