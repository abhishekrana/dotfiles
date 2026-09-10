//! Drawing the reader: the page body and one status row.

use ratatui::Frame;
use ratatui::layout::{Constraint, Layout};
use ratatui::style::Style as RStyle;
use ratatui::text::{Line, Span, Text};
use ratatui::widgets::Paragraph;
use unicode_width::UnicodeWidthStr;

use crate::app::App;
use crate::render;
use crate::theme::Role;

pub fn draw(frame: &mut Frame, app: &App) {
    let [body, status] = Layout::vertical([Constraint::Min(1), Constraint::Length(1)]).areas(frame.area());
    let page = app.page();
    let theme = app.theme();
    let rows = page
        .lines
        .iter()
        .skip(app.scroll())
        .take(usize::from(body.height))
        .map(|l| render::to_ratatui(l, theme, page.left))
        .collect::<Vec<_>>();
    frame.render_widget(Paragraph::new(Text::from(rows)), body);
    frame.render_widget(Paragraph::new(status_line(app, status.width)), status);
}

fn status_line(app: &App, width: u16) -> Line<'static> {
    let theme = app.theme();
    let bg = render::color(theme.color(Role::Surface));
    let fg = RStyle::default().fg(render::color(theme.color(Role::Fg))).bg(bg);
    let muted = RStyle::default().fg(render::color(theme.color(Role::Muted))).bg(bg);

    let name = app
        .buffer()
        .path()
        .and_then(|p| p.file_name())
        .map_or_else(|| "stdin".to_owned(), |n| n.to_string_lossy().into_owned());
    let section = current_section(app);
    let left = format!(" {name}  ");
    let theme_label = app.theme_label().map(|t| format!("{t}  ")).unwrap_or_default();
    let right = format!("{theme_label}{:>3}%  j/k scroll  T theme  q quit ", app.percent());
    let room = usize::from(width).saturating_sub(left.width() + right.width());
    let section = crate::layout::clip(&section, room);
    let gap = " ".repeat(room.saturating_sub(section.width()));
    Line::from(vec![
        Span::styled(left, fg),
        Span::styled(section, muted),
        Span::styled(gap, muted),
        Span::styled(right, muted),
    ])
}

/// Text of the last heading at or above the first row on screen.
fn current_section(app: &App) -> String {
    let Some(first) = app.page().lines.get(app.scroll()) else {
        return String::new();
    };
    app.doc()
        .headings
        .iter()
        .rev()
        .find(|h| h.block <= first.block)
        .map(|h| h.text.clone())
        .unwrap_or_default()
}
