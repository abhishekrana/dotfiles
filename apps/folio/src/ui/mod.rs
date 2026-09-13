//! Drawing the reader: the page body, overlays, and one bottom row.

use ratatui::Frame;
use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::style::{Color, Modifier, Style as RStyle};
use ratatui::text::{Line, Span, Text};
use ratatui::widgets::{Block, Borders, Clear, Paragraph};
use unicode_width::UnicodeWidthStr;

use crate::app::{App, Mode};
use crate::render;
use crate::theme::{Role, Theme};

const HELP: &[(&str, &str)] = &[
    ("j k  ↓ ↑", "scroll a line; the wheel does the same"),
    ("d u  PgDn PgUp  space", "half a page, a page"),
    ("g G  Home End", "top, bottom"),
    ("] [", "next, previous heading"),
    ("t", "outline; ↵ jumps, Esc closes"),
    ("/  n N", "search; next, previous match; Esc clears"),
    ("click", "follow a link"),
    ("drag", "select text; it goes to the clipboard on release"),
    ("Backspace", "back to the note a link was followed from"),
    ("y", "copy the selection, or the code block at the top of the screen"),
    ("e", "open the file in $EDITOR at the top line"),
    ("r", "reload the file"),
    ("w", "follow the file on disk and reload on change (on by default)"),
    ("T", "cycle the theme"),
    ("?", "this help"),
    ("q", "quit"),
];

pub fn draw(frame: &mut Frame, app: &App) {
    let [body, bottom] = Layout::vertical([Constraint::Min(1), Constraint::Length(1)]).areas(frame.area());
    draw_page(frame, body, app);
    match app.mode() {
        Mode::Outline { selected } => draw_outline(frame, body, app, *selected),
        Mode::Help => draw_help(frame, body, app.theme()),
        Mode::Read | Mode::Search { .. } => {}
    }
    let line = match app.mode() {
        Mode::Search { query } => search_line(app.theme(), query),
        _ => status_line(app, bottom.width),
    };
    frame.render_widget(Paragraph::new(line), bottom);
}

fn role(theme: &Theme, r: Role) -> Color {
    render::color(theme.color(r))
}

/// The page rows on screen, with the selection and search matches marked, the current match strongest.
fn draw_page(frame: &mut Frame, area: Rect, app: &App) {
    let page = app.page();
    let theme = app.theme();
    let mark = RStyle::default().bg(role(theme, Role::Selection));
    let current_mark = RStyle::default()
        .bg(role(theme, Role::Accent))
        .fg(role(theme, Role::Bg));
    let current = app.current_match().and_then(|i| app.matches().get(i)).copied();
    let rows = page
        .lines
        .iter()
        .enumerate()
        .skip(app.scroll())
        .take(usize::from(area.height))
        .map(|(i, l)| {
            let mut marks: Vec<(usize, usize, RStyle)> = app
                .matches()
                .iter()
                .filter(|m| m.row == i)
                .map(|m| (m.start, m.end, mark))
                .collect();
            if let Some((start, end)) = app.selection().and_then(|s| s.row_span(i)) {
                marks.push((start, end, mark));
            }
            if let Some(c) = current.filter(|c| c.row == i) {
                marks.push((c.start, c.end, current_mark));
            }
            render::to_ratatui_marked(l, theme, page.left, &marks)
        })
        .collect::<Vec<_>>();
    frame.render_widget(Paragraph::new(Text::from(rows)), area);
}

fn draw_outline(frame: &mut Frame, area: Rect, app: &App, selected: usize) {
    let theme = app.theme();
    let headings = &app.doc().headings;
    let width = headings
        .iter()
        .map(|h| h.text.width() + usize::from(h.level) * 2 + 4)
        .max()
        .unwrap_or(20)
        .clamp(24, 72);
    let height = (headings.len() + 2).clamp(3, usize::from(area.height).saturating_sub(2).max(3));
    let rect = centred(area, width as u16, height as u16);
    let inner_rows = usize::from(rect.height.saturating_sub(2)).max(1);
    let first = selected
        .saturating_sub(inner_rows - 1)
        .min(headings.len().saturating_sub(inner_rows));
    let fg = RStyle::default().fg(role(theme, Role::Fg));
    let muted = RStyle::default().fg(role(theme, Role::Muted));
    let picked = RStyle::default()
        .bg(role(theme, Role::Selection))
        .fg(role(theme, Role::Emphasis))
        .add_modifier(Modifier::BOLD);
    let rows: Vec<Line> = headings
        .iter()
        .enumerate()
        .skip(first)
        .take(inner_rows)
        .map(|(i, h)| {
            let indent = " ".repeat(usize::from(h.level.saturating_sub(1)) * 2);
            let text = format!(" {indent}{}", h.text);
            let pad = " ".repeat(usize::from(rect.width.saturating_sub(2)).saturating_sub(text.width()));
            let style = if i == selected {
                picked
            } else if h.level <= 2 {
                fg
            } else {
                muted
            };
            Line::from(Span::styled(format!("{text}{pad}"), style))
        })
        .collect();
    frame.render_widget(Clear, rect);
    frame.render_widget(Paragraph::new(rows).block(popup_block(theme, " Outline ")), rect);
}

fn draw_help(frame: &mut Frame, area: Rect, theme: &Theme) {
    let key_w = HELP.iter().map(|(k, _)| k.width()).max().unwrap_or(8);
    let rows: Vec<Line> = HELP
        .iter()
        .map(|(k, what)| {
            Line::from(vec![
                Span::styled(
                    format!(" {k:<key_w$}  "),
                    RStyle::default().fg(role(theme, Role::Accent)),
                ),
                Span::styled((*what).to_owned(), RStyle::default().fg(role(theme, Role::Fg))),
            ])
        })
        .collect();
    let width = rows.iter().map(Line::width).max().unwrap_or(40) as u16 + 3;
    let rect = centred(area, width, HELP.len() as u16 + 2);
    frame.render_widget(Clear, rect);
    frame.render_widget(Paragraph::new(rows).block(popup_block(theme, " Keys ")), rect);
}

fn popup_block(theme: &Theme, title: &'static str) -> Block<'static> {
    Block::default()
        .borders(Borders::ALL)
        .border_style(RStyle::default().fg(role(theme, Role::Border)))
        .title(Span::styled(
            title,
            RStyle::default()
                .fg(role(theme, Role::Emphasis))
                .add_modifier(Modifier::BOLD),
        ))
        .style(RStyle::default().bg(role(theme, Role::Bg)))
}

fn centred(area: Rect, width: u16, height: u16) -> Rect {
    let width = width.min(area.width);
    let height = height.min(area.height);
    Rect {
        x: area.x + (area.width - width) / 2,
        y: area.y + (area.height - height) / 2,
        width,
        height,
    }
}

fn search_line(theme: &Theme, query: &str) -> Line<'static> {
    let bg = role(theme, Role::Surface);
    Line::from(vec![
        Span::styled(" /", RStyle::default().fg(role(theme, Role::Accent)).bg(bg)),
        Span::styled(
            query.to_owned(),
            RStyle::default().fg(role(theme, Role::Emphasis)).bg(bg),
        ),
        Span::styled("▏", RStyle::default().fg(role(theme, Role::Accent)).bg(bg)),
    ])
}

fn status_line(app: &App, width: u16) -> Line<'static> {
    let theme = app.theme();
    let bg = role(theme, Role::Surface);
    let fg = RStyle::default().fg(role(theme, Role::Fg)).bg(bg);
    let muted = RStyle::default().fg(role(theme, Role::Muted)).bg(bg);
    let notice_style = RStyle::default().fg(role(theme, Role::Asking)).bg(bg);

    let name = app
        .buffer()
        .path()
        .and_then(|p| p.file_name())
        .map_or_else(|| "stdin".to_owned(), |n| n.to_string_lossy().into_owned());
    let left = format!(" {name}  ");
    let (middle, middle_style) = match app.notice() {
        Some(n) => (n.to_owned(), notice_style),
        None => (app.section().to_owned(), muted),
    };
    let theme_label = app.theme_label().map(|t| format!("{t}  ")).unwrap_or_default();
    let search = match (app.matches().len(), app.current_match()) {
        (n, Some(i)) if n > 0 => format!("{}/{n}  n N  ", i + 1),
        _ => String::new(),
    };
    let right = format!("{theme_label}{search}{:>3}%  t / ? q ", app.percent());
    let room = usize::from(width).saturating_sub(left.width() + right.width());
    let middle = crate::layout::clip(&middle, room);
    let gap = " ".repeat(room.saturating_sub(middle.width()));
    Line::from(vec![
        Span::styled(left, fg),
        Span::styled(middle, middle_style),
        Span::styled(gap, muted),
        Span::styled(right, muted),
    ])
}
