//! Table layout: fair-share column widths, then rows drawn as the style asks.

use unicode_width::UnicodeWidthStr;

use super::blocks::{Ctx, clip, fg};
use super::wrap::{words, wrap};
use super::{CellStyle, Line, Segment};
use crate::doc::{Align, Inline, Span};
use crate::style::{Style, TableLines};
use crate::theme::Role;

/// A column never shrinks below this many cells.
const MIN_COL: usize = 3;

pub(super) fn layout_table(
    align: &[Align],
    head: &[Vec<Inline>],
    rows: &[Vec<Vec<Inline>>],
    width: usize,
    style: &Style,
    cx: Ctx,
    span: Span,
) -> Vec<Line> {
    let t = &style.table;
    let ncols = head.len().max(rows.iter().map(Vec::len).max().unwrap_or(0));
    if ncols == 0 {
        return Vec::new();
    }
    let header_style = CellStyle {
        fg: Some(Role::Emphasis),
        bold: t.header_bold,
        ..cx.base
    };
    let body_style = CellStyle {
        fg: cx.base.fg.or(Some(style.paragraph.fg)),
        ..cx.base
    };
    let head_cells: Vec<Vec<Segment>> = (0..ncols).map(|i| cell(head.get(i), header_style, style)).collect();
    let body_cells: Vec<Vec<Vec<Segment>>> = rows
        .iter()
        .map(|r| (0..ncols).map(|i| cell(r.get(i), body_style, style)).collect())
        .collect();

    let mut widths: Vec<usize> = (0..ncols)
        .map(|i| {
            let h = row_width(&head_cells[i]);
            body_cells.iter().map(|r| row_width(&r[i])).fold(h, usize::max)
        })
        .collect();
    let chrome = match t.lines {
        TableLines::Box => 3 * ncols + 1,
        TableLines::Rules | TableLines::None => 3 * (ncols - 1),
    };
    let available = width.saturating_sub(chrome).max(ncols * MIN_COL);
    shrink(&mut widths, available);
    let table_width = widths.iter().sum::<usize>() + chrome;
    let lines_fg = fg(t.line_fg);

    let mut out = Vec::new();
    let mut emit = |segs: Vec<Segment>| {
        out.push(Line {
            segments: segs,
            block: 0,
            src: Some(span),
        });
    };
    let bar = |s: &str| Segment::new(s.to_owned(), lines_fg);
    let horizontal = |left: &str, mid: &str, right: &str| -> Vec<Segment> {
        let inner = widths.iter().map(|w| "─".repeat(w + 2)).collect::<Vec<_>>().join(mid);
        vec![Segment::new(format!("{left}{inner}{right}"), lines_fg)]
    };
    let row_segs = |cells: &[Vec<Segment>], bg: Option<Role>| -> Vec<Segment> {
        let mut segs = Vec::new();
        for (i, c) in cells.iter().enumerate() {
            if t.lines == TableLines::Box {
                segs.push(bar(if i == 0 { "│ " } else { " │ " }));
            } else if i > 0 {
                segs.push(Segment::new(
                    "   ".to_owned(),
                    CellStyle {
                        bg,
                        ..CellStyle::default()
                    },
                ));
            }
            segs.extend(padded(c, widths[i], align.get(i).copied().unwrap_or(Align::Left), bg));
        }
        if t.lines == TableLines::Box {
            segs.push(bar(" │"));
        }
        segs
    };

    if t.lines == TableLines::Box {
        emit(horizontal("┌", "┬", "┐"));
    }
    emit(row_segs(&head_cells, None));
    if t.header_rule {
        emit(match t.lines {
            TableLines::Box => horizontal("├", "┼", "┤"),
            _ => vec![Segment::new("─".repeat(table_width), lines_fg)],
        });
    }
    for (i, r) in body_cells.iter().enumerate() {
        let bg = t.zebra.filter(|_| i % 2 == 1);
        emit(row_segs(r, bg));
        if t.lines == TableLines::Rules && i + 1 < body_cells.len() {
            emit(vec![Segment::new("─".repeat(table_width), lines_fg)]);
        }
    }
    if t.lines == TableLines::Box {
        emit(horizontal("└", "┴", "┘"));
    }
    out
}

/// One cell as a single row of segments; cells never wrap.
fn cell(inlines: Option<&Vec<Inline>>, base: CellStyle, style: &Style) -> Vec<Segment> {
    let Some(inlines) = inlines else { return Vec::new() };
    wrap(&words(inlines, base, style), usize::MAX / 2)
        .into_iter()
        .next()
        .unwrap_or_default()
}

fn row_width(segs: &[Segment]) -> usize {
    segs.iter().map(Segment::width).sum()
}

/// Takes cells from the widest column first until the table fits.
fn shrink(widths: &mut [usize], available: usize) {
    while widths.iter().sum::<usize>() > available {
        let Some((i, _)) = widths
            .iter()
            .enumerate()
            .filter(|(_, w)| **w > MIN_COL)
            .max_by_key(|(_, w)| **w)
        else {
            return;
        };
        widths[i] -= 1;
    }
}

/// Clips a cell to its column and pads it according to the alignment.
fn padded(segs: &[Segment], width: usize, align: Align, bg: Option<Role>) -> Vec<Segment> {
    let mut out: Vec<Segment> = Vec::new();
    let mut used = 0;
    for s in segs {
        let w = s.width();
        let text = if used + w > width {
            clip(&s.text, width - used)
        } else {
            s.text.clone()
        };
        used += text.width();
        out.push(Segment {
            text,
            style: CellStyle {
                bg: bg.or(s.style.bg),
                ..s.style
            },
            src: s.src,
            link: s.link.clone(),
        });
        if used >= width {
            break;
        }
    }
    let gap = width.saturating_sub(used);
    let (before, after) = match align {
        Align::Left => (0, gap),
        Align::Center => (gap / 2, gap - gap / 2),
        Align::Right => (gap, 0),
    };
    let pad = |n: usize| {
        Segment::new(
            " ".repeat(n),
            CellStyle {
                bg,
                ..CellStyle::default()
            },
        )
    };
    if before > 0 {
        out.insert(0, pad(before));
    }
    if after > 0 {
        out.push(pad(after));
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn widest_column_gives_way_first() {
        let mut w = vec![4, 20, 6];
        shrink(&mut w, 20);
        assert_eq!(w, vec![4, 10, 6]);
    }

    #[test]
    fn shrinking_stops_at_the_minimum() {
        let mut w = vec![3, 3];
        shrink(&mut w, 2);
        assert_eq!(w, vec![3, 3]);
    }
}
