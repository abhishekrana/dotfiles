//! Mouse selection over the laid-out page: a range in page cells and the text it covers.

use unicode_segmentation::UnicodeSegmentation;
use unicode_width::UnicodeWidthStr;

use crate::layout::Page;

/// A cell of the page: a row, and a column within the reading column.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub struct Spot {
    pub row: usize,
    pub col: usize,
}

/// A drag in progress or finished: where it started and where the pointer is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Range {
    pub anchor: Spot,
    pub cursor: Spot,
}

impl Range {
    #[must_use]
    pub fn new(anchor: Spot) -> Self {
        Self { anchor, cursor: anchor }
    }

    /// The ends in page order.
    #[must_use]
    pub fn ends(&self) -> (Spot, Spot) {
        if self.anchor <= self.cursor {
            (self.anchor, self.cursor)
        } else {
            (self.cursor, self.anchor)
        }
    }

    /// A drag that never moved is a click, not a selection.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.anchor == self.cursor
    }

    /// The cells `row` contributes, or `None` when the row is outside the range.
    /// The end is `usize::MAX` on every row but the last, so the highlight runs to the end of the text.
    #[must_use]
    pub fn row_span(&self, row: usize) -> Option<(usize, usize)> {
        let (from, to) = self.ends();
        if row < from.row || row > to.row {
            return None;
        }
        let start = if row == from.row { from.col } else { 0 };
        let end = if row == to.row { to.col } else { usize::MAX };
        (start < end).then_some((start, end))
    }

    /// The selected text, one line per page row, trailing padding trimmed.
    #[must_use]
    pub fn text(&self, page: &Page) -> String {
        let (from, to) = self.ends();
        let mut out = String::new();
        for row in from.row..=to.row {
            let Some((start, end)) = self.row_span(row) else {
                continue;
            };
            let line: String = page
                .lines
                .get(row)
                .map(|l| l.segments.iter().map(|s| s.text.as_str()).collect())
                .unwrap_or_default();
            if row > from.row {
                out.push('\n');
            }
            out.push_str(slice(&line, start, end).trim_end());
        }
        out
    }
}

/// The part of `text` covering cell columns `start..end`.
fn slice(text: &str, start: usize, end: usize) -> String {
    let mut out = String::new();
    let mut col = 0;
    for g in text.graphemes(true) {
        if col >= end {
            break;
        }
        if col >= start {
            out.push_str(g);
        }
        col += g.width();
    }
    out
}

/// The cell a pointer at `col` sits on, given the page's left margin.
#[must_use]
pub fn column(page: &Page, col: u16) -> usize {
    usize::from(col).saturating_sub(usize::from(page.left))
}

/// Cells in the widest row, used to clamp a pointer past the end of a line.
#[must_use]
pub fn row_width(page: &Page, row: usize) -> usize {
    page.lines
        .get(row)
        .map_or(0, |l| l.segments.iter().map(|s| s.text.width()).sum::<usize>())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn spot(row: usize, col: usize) -> Spot {
        Spot { row, col }
    }

    #[test]
    fn a_drag_that_never_moved_is_a_click() {
        assert!(Range::new(spot(2, 4)).is_empty());
    }

    #[test]
    fn rows_between_the_ends_run_to_the_end_of_the_text() {
        let r = Range {
            anchor: spot(1, 3),
            cursor: spot(3, 5),
        };
        assert_eq!(r.row_span(0), None);
        assert_eq!(r.row_span(1), Some((3, usize::MAX)));
        assert_eq!(r.row_span(2), Some((0, usize::MAX)));
        assert_eq!(r.row_span(3), Some((0, 5)));
        assert_eq!(r.row_span(4), None);
    }

    #[test]
    fn a_backwards_drag_selects_the_same_cells() {
        let forward = Range {
            anchor: spot(1, 2),
            cursor: spot(2, 6),
        };
        let backward = Range {
            anchor: spot(2, 6),
            cursor: spot(1, 2),
        };
        assert_eq!(forward.ends(), backward.ends());
        assert_eq!(forward.row_span(1), backward.row_span(1));
    }

    #[test]
    fn slicing_counts_cells_not_bytes() {
        assert_eq!(slice("héllo wörld", 2, 7), "llo w");
        assert_eq!(slice("日本語です", 2, 6), "本語");
        assert_eq!(slice("short", 0, usize::MAX), "short");
    }
}
