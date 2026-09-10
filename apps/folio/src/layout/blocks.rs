//! Block layout: each block kind to rows at a width, following the style's rules.

use unicode_width::UnicodeWidthStr;

use super::table::layout_table;
use super::wrap::{footnote_marker, words, wrap};
use super::{CellStyle, Line, Segment};
use crate::doc::{Block, CalloutKind, Inline, ListItem, Span};
use crate::highlight::Highlighter;
use crate::style::{CodeLabel, FrontMatterAs, HeadingLine, HrWidth, Style};
use crate::theme::Role;

/// What the enclosing blocks pass down: nesting depth, the text style they impose, and the highlighter.
#[derive(Debug, Clone, Copy)]
pub(super) struct Ctx<'a> {
    pub depth: u8,
    pub base: CellStyle,
    pub hl: &'a Highlighter,
}

impl<'a> Ctx<'a> {
    pub(super) fn new(hl: &'a Highlighter) -> Self {
        Self {
            depth: 0,
            base: CellStyle::default(),
            hl,
        }
    }
}

/// Blank rows the style wants before and after a block.
pub(super) fn spacing(block: &Block, style: &Style) -> (u8, u8) {
    match block {
        Block::Heading { level, .. } => (style.heading(*level).above, style.heading(*level).below),
        Block::Code { .. } => (style.code.above, style.code.below),
        Block::Footnote { .. } => (0, 0),
        _ => (0, style.paragraph.below),
    }
}

pub(super) fn layout_block(block: &Block, width: usize, style: &Style, cx: Ctx) -> Vec<Line> {
    match block {
        Block::FrontMatter { fields, span } => front_matter(fields, width, style, cx, *span),
        Block::Heading {
            level, inlines, span, ..
        } => heading(*level, inlines, width, style, cx, *span),
        Block::Paragraph { inlines, span } => paragraph(inlines, width, style, cx, *span),
        Block::List {
            ordered,
            start,
            tight,
            items,
            ..
        } => list(*ordered, *start, *tight, items, width, style, cx),
        Block::Quote { blocks, .. } => quote(blocks, width, style, cx),
        Block::Callout {
            kind, title, blocks, ..
        } => callout(*kind, title.as_deref(), blocks, width, style, cx),
        Block::Code { lang, text, span } => code(lang.as_deref(), text, width, style, cx, *span),
        Block::Table {
            align,
            head,
            rows,
            span,
        } => layout_table(align, head, rows, width, style, cx, *span),
        Block::Rule { span } => rule(width, style, *span),
        Block::Footnote { label, blocks, .. } => footnote(label, blocks, width, style, cx),
        Block::Image { alt, src, span } => image(alt, src, width, style, *span),
    }
}

/// Lays out a run of blocks; `spaced` puts the style's blank rows between them, tight lists do not.
pub(super) fn layout_blocks(blocks: &[Block], width: usize, style: &Style, cx: Ctx, spaced: bool) -> Vec<Line> {
    let mut out = Vec::new();
    for (i, block) in blocks.iter().enumerate() {
        if spaced && i > 0 {
            let (above, _) = spacing(block, style);
            let (_, below) = spacing(&blocks[i - 1], style);
            for _ in 0..above.max(below) {
                out.push(Line::default());
            }
        }
        out.extend(layout_block(block, width, style, cx));
    }
    out
}

fn line(segments: Vec<Segment>, src: Option<Span>) -> Line {
    Line {
        segments,
        block: 0,
        src,
    }
}

fn text_rows(inlines: &[Inline], width: usize, style: &Style, base: CellStyle, span: Span) -> Vec<Line> {
    wrap(&words(inlines, base, style), width)
        .into_iter()
        .map(|segs| line(segs, Some(span)))
        .collect()
}

fn heading(level: u8, inlines: &[Inline], width: usize, style: &Style, cx: Ctx, span: Span) -> Vec<Line> {
    let rule = style.heading(level);
    let base = CellStyle {
        fg: Some(rule.fg),
        bold: rule.bold,
        italic: rule.italic,
        ..cx.base
    };
    let mut rows = text_rows(inlines, width, style, base, span);
    let underline = match rule.rule {
        HeadingLine::None => 0,
        HeadingLine::Words => rows.iter().map(Line::width).max().unwrap_or(0),
        HeadingLine::Column => width,
    };
    if underline > 0 {
        rows.push(line(
            vec![Segment::new(style.rule.glyph.repeat(underline), fg(style.rule.fg))],
            Some(span),
        ));
    }
    rows
}

fn paragraph(inlines: &[Inline], width: usize, style: &Style, cx: Ctx, span: Span) -> Vec<Line> {
    let base = CellStyle {
        fg: cx.base.fg.or(Some(style.paragraph.fg)),
        ..cx.base
    };
    text_rows(inlines, width, style, base, span)
}

fn list(
    ordered: bool,
    start: usize,
    tight: bool,
    items: &[ListItem],
    width: usize,
    style: &Style,
    cx: Ctx,
) -> Vec<Line> {
    let mut out = Vec::new();
    let marker_width = if ordered {
        format!("{}.", start + items.len()).width()
    } else {
        0
    };
    for (i, item) in items.iter().enumerate() {
        let (marker, marker_style, text_base) = marker(ordered, start + i, marker_width, item, style, cx);
        let indent = marker.width() + 1;
        let inner_width = width.saturating_sub(indent).max(1);
        let inner = Ctx {
            depth: cx.depth + 1,
            base: text_base,
            ..cx
        };
        if !tight && i > 0 {
            out.push(Line::default());
        }
        let rows = layout_blocks(&item.blocks, inner_width, style, inner, !tight);
        for (n, mut row) in rows.into_iter().enumerate() {
            let prefix = if n == 0 {
                vec![Segment::new(format!("{marker} "), marker_style)]
            } else {
                vec![Segment::new(" ".repeat(indent), CellStyle::default())]
            };
            row.segments.splice(0..0, prefix);
            out.push(row);
        }
    }
    out
}

fn marker(
    ordered: bool,
    n: usize,
    marker_width: usize,
    item: &ListItem,
    style: &Style,
    cx: Ctx,
) -> (String, CellStyle, CellStyle) {
    if let Some(done) = item.task {
        let t = &style.task;
        let glyph = if done { t.done.clone() } else { t.todo.clone() };
        let marker_fg = if done { t.done_fg } else { t.todo_fg };
        let text = if done {
            CellStyle {
                fg: Some(t.done_text),
                ..cx.base
            }
        } else {
            cx.base
        };
        return (glyph, fg(marker_fg), text);
    }
    if ordered {
        let label = format!("{n}.");
        return (format!("{label:>marker_width$}"), fg(style.list.bullet_fg), cx.base);
    }
    let glyph = if cx.depth == 0 {
        &style.list.bullet
    } else {
        &style.list.nested
    };
    (glyph.clone(), fg(style.list.bullet_fg), cx.base)
}

fn quote(blocks: &[Block], width: usize, style: &Style, cx: Ctx) -> Vec<Line> {
    let q = &style.quote;
    let base = CellStyle {
        fg: Some(q.fg),
        italic: cx.base.italic || q.italic,
        ..cx.base
    };
    let bar = Segment::new(format!("{} ", q.bar), fg(q.bar_fg));
    let inner_width = width.saturating_sub(bar.width()).max(1);
    layout_blocks(blocks, inner_width, style, Ctx { base, ..cx }, true)
        .into_iter()
        .map(|mut row| {
            row.segments.insert(0, bar.clone());
            row
        })
        .collect()
}

fn callout(
    kind: CalloutKind,
    title: Option<&str>,
    blocks: &[Block],
    width: usize,
    style: &Style,
    cx: Ctx,
) -> Vec<Line> {
    let c = &style.callout;
    let accent = match kind {
        CalloutKind::Note => c.kinds.note,
        CalloutKind::Tip => c.kinds.tip,
        CalloutKind::Important => c.kinds.important,
        CalloutKind::Warning => c.kinds.warning,
        CalloutKind::Caution => c.kinds.caution,
    };
    let bar = Segment::new(
        format!("{} ", c.bar),
        CellStyle {
            fg: Some(accent),
            bg: c.bg,
            ..CellStyle::default()
        },
    );
    let inner_width = width.saturating_sub(bar.width()).max(1);
    let mut title_segs = vec![bar.clone()];
    if c.icon {
        title_segs.push(Segment::new(
            format!("{} ", icon(kind, &c.icons)),
            CellStyle {
                fg: Some(accent),
                bg: c.bg,
                ..CellStyle::default()
            },
        ));
    }
    title_segs.push(Segment::new(
        title.unwrap_or(kind.title()).to_owned(),
        CellStyle {
            fg: Some(accent),
            bg: c.bg,
            bold: true,
            ..CellStyle::default()
        },
    ));
    let mut rows = vec![line(title_segs, None)];
    let base = CellStyle {
        bg: c.bg.or(cx.base.bg),
        ..cx.base
    };
    for mut row in layout_blocks(blocks, inner_width, style, Ctx { base, ..cx }, true) {
        row.segments.insert(0, bar.clone());
        rows.push(row);
    }
    if c.bg.is_some() {
        for row in &mut rows {
            fill(row, width, c.bg);
        }
    }
    rows
}

fn icon(kind: CalloutKind, icons: &crate::style::CalloutIcons) -> &str {
    match kind {
        CalloutKind::Note => &icons.note,
        CalloutKind::Tip => &icons.tip,
        CalloutKind::Important => &icons.important,
        CalloutKind::Warning => &icons.warning,
        CalloutKind::Caution => &icons.caution,
    }
}

fn code(lang: Option<&str>, text: &str, width: usize, style: &Style, cx: Ctx, span: Span) -> Vec<Line> {
    let c = &style.code;
    let pad = usize::from(c.pad);
    let inner = width.saturating_sub(2 * pad).max(1);
    let body = CellStyle {
        fg: Some(style.paragraph.fg),
        bg: c.bg,
        ..CellStyle::default()
    };
    let label = lang
        .filter(|_| c.label != CodeLabel::None)
        .map(|l| Segment::new(l.to_owned(), fg(Role::Muted)));
    let mut rows = Vec::new();
    if let (CodeLabel::Above, Some(label)) = (c.label, label.clone()) {
        rows.push(line(vec![label], Some(span)));
    }
    let mut top = line(Vec::new(), Some(span));
    if let (CodeLabel::Right, Some(label)) = (c.label, label) {
        let gap = width.saturating_sub(label.width() + pad);
        top.segments.push(Segment::new(" ".repeat(gap), body));
        top.segments.push(Segment {
            style: CellStyle {
                bg: c.bg,
                ..label.style
            },
            ..label
        });
    }
    fill(&mut top, width, c.bg);
    rows.push(top);
    let expanded = text.replace('\t', "    ");
    let code_rows: Vec<Vec<Segment>> = match lang.and_then(|l| cx.hl.highlight(l, &expanded)) {
        Some(lines) => lines
            .into_iter()
            .map(|tokens| {
                tokens
                    .into_iter()
                    .map(|t| {
                        Segment::new(
                            t.text,
                            CellStyle {
                                fg_rgb: t.fg,
                                bold: t.bold,
                                italic: t.italic,
                                underline: t.underline,
                                ..body
                            },
                        )
                    })
                    .collect()
            })
            .collect(),
        None => expanded
            .lines()
            .map(|l| vec![Segment::new(l.to_owned(), body)])
            .collect(),
    };
    for segs in code_rows {
        let mut row = line(vec![Segment::new(" ".repeat(pad), body)], Some(span));
        row.segments.extend(clip_segments(segs, inner, body));
        fill(&mut row, width, c.bg);
        rows.push(row);
    }
    let mut bottom = line(Vec::new(), Some(span));
    fill(&mut bottom, width, c.bg);
    rows.push(bottom);
    rows
}

/// Cuts a row of segments to `width` cells, ending it with `→` when something was cut.
pub(super) fn clip_segments(segs: Vec<Segment>, width: usize, marker: CellStyle) -> Vec<Segment> {
    if segs.iter().map(Segment::width).sum::<usize>() <= width {
        return segs;
    }
    let limit = width.saturating_sub(1);
    let mut out = Vec::new();
    let mut used = 0;
    for mut s in segs {
        let w = s.width();
        if used + w > limit {
            s.text = take_cells(&s.text, limit - used);
            if !s.text.is_empty() {
                out.push(s);
            }
            break;
        }
        used += w;
        out.push(s);
    }
    out.push(Segment::new("→", marker));
    out
}

fn rule(width: usize, style: &Style, span: Span) -> Vec<Line> {
    let r = &style.rule;
    let segs = match r.width {
        HrWidth::Column => {
            let glyph_w = r.glyph.width().max(1);
            vec![Segment::new(r.glyph.repeat(width / glyph_w), fg(r.fg))]
        }
        HrWidth::Glyph => {
            let left = width.saturating_sub(r.glyph.width()) / 2;
            vec![
                Segment::new(" ".repeat(left), CellStyle::default()),
                Segment::new(r.glyph.clone(), fg(r.fg)),
            ]
        }
    };
    vec![line(segs, Some(span))]
}

fn front_matter(fields: &[(String, String)], width: usize, style: &Style, cx: Ctx, span: Span) -> Vec<Line> {
    let muted = fg(Role::Muted);
    match style.front_matter.as_ {
        FrontMatterAs::Hidden => Vec::new(),
        FrontMatterAs::Line => {
            let text = fields
                .iter()
                .map(|(k, v)| format!("{k} {v}"))
                .collect::<Vec<_>>()
                .join("  ·  ");
            vec![line(vec![Segment::new(clip(&text, width), muted)], Some(span))]
        }
        FrontMatterAs::List => {
            let kw = fields.iter().map(|(k, _)| k.width()).max().unwrap_or(0);
            fields
                .iter()
                .map(|(k, v)| {
                    let key = Segment::new(format!("{k:<kw$}  "), muted);
                    let value = Segment::new(clip(v, width.saturating_sub(kw + 2)), fg(style.paragraph.fg));
                    line(vec![key, value], Some(span))
                })
                .collect()
        }
        FrontMatterAs::Table => {
            let head = vec![
                vec![Inline::Text("key".into(), span)],
                vec![Inline::Text("value".into(), span)],
            ];
            let rows = fields
                .iter()
                .map(|(k, v)| vec![vec![Inline::Text(k.clone(), span)], vec![Inline::Text(v.clone(), span)]])
                .collect::<Vec<_>>();
            layout_table(&[], &head, &rows, width, style, Ctx::new(cx.hl), span)
        }
    }
}

fn footnote(label: &str, blocks: &[Block], width: usize, style: &Style, cx: Ctx) -> Vec<Line> {
    let marker = Segment::new(format!("{} ", footnote_marker(label, style)), fg(style.footnote.fg));
    let indent = marker.width();
    let base = CellStyle {
        fg: Some(style.footnote.text_fg),
        ..cx.base
    };
    layout_blocks(
        blocks,
        width.saturating_sub(indent).max(1),
        style,
        Ctx { base, ..cx },
        true,
    )
    .into_iter()
    .enumerate()
    .map(|(n, mut row)| {
        let prefix = if n == 0 {
            marker.clone()
        } else {
            Segment::new(" ".repeat(indent), CellStyle::default())
        };
        row.segments.insert(0, prefix);
        row
    })
    .collect()
}

fn image(alt: &str, src: &str, width: usize, style: &Style, span: Span) -> Vec<Line> {
    let text = format!("{} {alt} ({src})", style.image.placeholder);
    vec![line(
        vec![Segment::new(clip(&text, width), fg(style.image.fg))],
        Some(span),
    )]
}

/// Pads a row with spaces to `width`, in `bg`, so a tinted block fills its column.
pub(super) fn fill(row: &mut Line, width: usize, bg: Option<Role>) {
    let used = row.width();
    if used < width {
        row.segments.push(Segment::new(
            " ".repeat(width - used),
            CellStyle {
                bg,
                ..CellStyle::default()
            },
        ));
    }
}

/// Truncates to `width` cells with an ellipsis when the text is longer.
#[must_use]
pub fn clip(text: &str, width: usize) -> String {
    if text.width() <= width {
        return text.to_owned();
    }
    let mut out = take_cells(text, width.saturating_sub(1));
    out.push('…');
    out
}

/// The longest prefix that fits in `width` cells.
fn take_cells(text: &str, width: usize) -> String {
    let mut out = String::new();
    let mut used = 0;
    for ch in text.chars() {
        let w = unicode_width::UnicodeWidthChar::width(ch).unwrap_or(0);
        if used + w > width {
            break;
        }
        out.push(ch);
        used += w;
    }
    out
}

pub(super) fn fg(role: Role) -> CellStyle {
    CellStyle {
        fg: Some(role),
        ..CellStyle::default()
    }
}
