//! Layout: document × style × width -> lines of styled segments, each mapped back to its source span.

mod blocks;
mod table;
mod wrap;

pub use blocks::clip;

use std::collections::HashMap;
use std::hash::{Hash, Hasher};
use std::sync::Arc;

use tracing::debug;
use unicode_width::UnicodeWidthStr;

use crate::doc::{Block, Document, Span};
use crate::highlight::Highlighter;
use crate::style::{Align, Measure, Style};
use crate::theme::{Rgb, Role, Theme};

/// The smallest measure a pane can force before text stops being readable at all.
const MIN_MEASURE: u16 = 16;
/// Cells kept clear on the left of the column, and on each side when the pane is narrower than the measure.
const GUTTER: u16 = 2;

/// Presentation of one run of cells. Colours are roles; the theme resolves them at paint time.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Hash)]
pub struct CellStyle {
    pub fg: Option<Role>,
    /// Direct colour from a syntax theme, for highlighted code; wins over `fg`.
    pub fg_rgb: Option<Rgb>,
    pub bg: Option<Role>,
    pub bold: bool,
    pub italic: bool,
    pub underline: bool,
    pub strike: bool,
}

/// A run of text in one style, with the source it came from and the link it belongs to.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Segment {
    pub text: String,
    pub style: CellStyle,
    pub src: Option<Span>,
    pub link: Option<Arc<str>>,
}

impl Segment {
    #[must_use]
    pub fn new(text: impl Into<String>, style: CellStyle) -> Self {
        Self {
            text: text.into(),
            style,
            src: None,
            link: None,
        }
    }

    #[must_use]
    pub fn width(&self) -> usize {
        self.text.width()
    }
}

/// One screen row of the page.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Line {
    pub segments: Vec<Segment>,
    /// Index of the block this row belongs to.
    pub block: usize,
    /// Source range the row was laid out from, when it has one.
    pub src: Option<Span>,
}

impl Line {
    #[must_use]
    pub fn width(&self) -> usize {
        self.segments.iter().map(Segment::width).sum()
    }

    #[must_use]
    pub fn is_blank(&self) -> bool {
        self.segments.iter().all(|s| s.text.trim().is_empty())
    }
}

/// The laid-out document at one pane width.
#[derive(Debug, Clone, Default)]
pub struct Page {
    pub lines: Vec<Line>,
    /// Reading column width in cells.
    pub measure: u16,
    /// Cells of margin before the column.
    pub left: u16,
    pub width: u16,
}

/// Lays out documents, caching each block's rows per measure.
#[derive(Debug)]
pub struct Layouter {
    cache: HashMap<(u64, u16), Vec<Line>>,
    highlighter: Highlighter,
}

impl Layouter {
    #[must_use]
    pub fn new(theme: &Theme) -> Self {
        Self {
            cache: HashMap::new(),
            highlighter: Highlighter::for_theme(theme),
        }
    }

    /// Drops every cached row; call when the style changes.
    pub fn clear(&mut self) {
        self.cache.clear();
    }

    /// Switches the code highlighting theme; cached rows carry colours, so they are dropped.
    pub fn set_theme(&mut self, theme: &Theme) {
        self.highlighter = Highlighter::for_theme(theme);
        self.clear();
    }

    pub fn layout(&mut self, doc: &Document, style: &Style, width: u16) -> Page {
        let fit = width.saturating_sub(2 * GUTTER);
        let measure = match style.measure {
            Measure::Full(_) => fit,
            Measure::Cells(n) => n.min(fit),
        }
        .max(MIN_MEASURE.min(width));
        let left = match style.align {
            Align::Left => GUTTER.min(width.saturating_sub(measure)),
            Align::Center => width.saturating_sub(measure) / 2,
        };
        let mut lines = Vec::new();
        let mut pending_blank: u8 = 0;
        let mut hits = 0usize;
        for (index, block) in doc.blocks.iter().enumerate() {
            let (above, below) = blocks::spacing(block, style);
            if !lines.is_empty() {
                for _ in 0..pending_blank.max(above) {
                    lines.push(Line {
                        block: index,
                        ..Line::default()
                    });
                }
            }
            let key = (hash_block(block), measure);
            let rows = if let Some(rows) = self.cache.get(&key) {
                hits += 1;
                rows.clone()
            } else {
                let mut rows =
                    blocks::layout_block(block, usize::from(measure), style, blocks::Ctx::new(&self.highlighter));
                for row in &mut rows {
                    if row.width() > usize::from(measure) {
                        let segs = std::mem::take(&mut row.segments);
                        row.segments = blocks::clip_segments(segs, usize::from(measure), blocks::fg(Role::Muted));
                    }
                }
                self.cache.insert(key, rows.clone());
                rows
            };
            lines.extend(rows.into_iter().map(|mut l| {
                l.block = index;
                l
            }));
            pending_blank = below;
        }
        debug!(width, measure, lines = lines.len(), cache_hits = hits, "layout");
        Page {
            lines,
            measure,
            left,
            width,
        }
    }
}

fn hash_block(block: &Block) -> u64 {
    let mut h = std::hash::DefaultHasher::new();
    block.hash(&mut h);
    h.finish()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::buffer::Buffer;

    fn page(text: &str, width: u16) -> Page {
        let doc = crate::doc::parse(&Buffer::from_text(text));
        let style = crate::style::load("github", None).expect("style");
        let theme = Theme::default_theme().expect("theme");
        Layouter::new(theme).layout(&doc, &style, width)
    }

    #[test]
    fn measure_fills_the_pane_inside_the_gutters() {
        let p = page("Hello\n", 100);
        assert_eq!((p.measure, p.left), (96, 2));
        let narrow = page("Hello\n", 50);
        assert_eq!((narrow.measure, narrow.left), (46, 2));
    }

    #[test]
    fn blank_rows_collapse_between_blocks() {
        let p = page("# A\n\nText.\n\n## B\n", 100);
        let blanks = p.lines.iter().filter(|l| l.is_blank()).count();
        // A's rule 0 below · Text 1 below vs B 1 above -> 1 · nothing after B
        assert_eq!(
            blanks,
            1,
            "{:?}",
            p.lines
                .iter()
                .map(|l| l.segments.iter().map(|s| s.text.as_str()).collect::<String>())
                .collect::<Vec<_>>()
        );
    }

    #[test]
    fn no_row_exceeds_the_measure() {
        let long = "word ".repeat(60);
        let p = page(&format!("# T\n\n{long}\n\n| a | b |\n|---|---|\n| {long} | x |\n"), 90);
        assert!(
            p.lines.iter().all(|l| l.width() <= usize::from(p.measure)),
            "a row overflowed"
        );
    }
}
