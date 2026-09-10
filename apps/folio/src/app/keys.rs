//! Key bindings: vi and less, nothing to learn.

use crossterm::event::{KeyCode, KeyEvent, KeyEventKind, KeyModifiers};

use super::Msg;

pub(super) fn map(key: KeyEvent) -> Option<Msg> {
    if key.kind == KeyEventKind::Release {
        return None;
    }
    let ctrl = key.modifiers.contains(KeyModifiers::CONTROL);
    Some(match key.code {
        KeyCode::Char('j') | KeyCode::Down => Msg::ScrollLines(1),
        KeyCode::Char('k') | KeyCode::Up => Msg::ScrollLines(-1),
        KeyCode::Char('d') | KeyCode::PageDown => Msg::HalfPage(1),
        KeyCode::Char('u') | KeyCode::PageUp => Msg::HalfPage(-1),
        KeyCode::Char(' ') => Msg::HalfPage(2),
        KeyCode::Char('g') | KeyCode::Home => Msg::Top,
        KeyCode::Char('G') | KeyCode::End => Msg::Bottom,
        KeyCode::Char('q' | 'Q') => Msg::Quit,
        KeyCode::Char('c') if ctrl => Msg::Quit,
        _ => return None,
    })
}
