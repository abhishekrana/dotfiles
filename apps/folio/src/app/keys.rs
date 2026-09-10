//! Key bindings per mode: vi and less in the reader, plain text entry in search and hints.

use crossterm::event::{KeyCode, KeyEvent, KeyEventKind, KeyModifiers};

use super::{Mode, Msg};

pub(super) fn map(key: KeyEvent, mode: &Mode) -> Option<Msg> {
    if key.kind == KeyEventKind::Release {
        return None;
    }
    let ctrl = key.modifiers.contains(KeyModifiers::CONTROL);
    if ctrl && key.code == KeyCode::Char('c') {
        return Some(Msg::Quit);
    }
    match mode {
        Mode::Read => read(key.code),
        Mode::Outline { .. } => Some(match key.code {
            KeyCode::Char('j') | KeyCode::Down => Msg::Down,
            KeyCode::Char('k') | KeyCode::Up => Msg::Up,
            KeyCode::Enter => Msg::Select,
            KeyCode::Esc | KeyCode::Char('t') => Msg::Cancel,
            KeyCode::Char('q') => Msg::Quit,
            _ => return None,
        }),
        Mode::Search { .. } | Mode::Hints { .. } => Some(match key.code {
            KeyCode::Esc => Msg::Cancel,
            KeyCode::Enter => Msg::Select,
            KeyCode::Backspace => Msg::Backspace,
            KeyCode::Char(c) if !ctrl => Msg::Input(c),
            _ => return None,
        }),
        Mode::Help => Some(if key.code == KeyCode::Char('q') {
            Msg::Quit
        } else {
            Msg::Cancel
        }),
    }
}

fn read(code: KeyCode) -> Option<Msg> {
    Some(match code {
        KeyCode::Char('j') | KeyCode::Down => Msg::ScrollLines(1),
        KeyCode::Char('k') | KeyCode::Up => Msg::ScrollLines(-1),
        KeyCode::Char('d') | KeyCode::PageDown => Msg::HalfPage(1),
        KeyCode::Char('u') | KeyCode::PageUp => Msg::HalfPage(-1),
        KeyCode::Char(' ') => Msg::HalfPage(2),
        KeyCode::Char('g') | KeyCode::Home => Msg::Top,
        KeyCode::Char('G') | KeyCode::End => Msg::Bottom,
        KeyCode::Char(']') => Msg::NextHeading,
        KeyCode::Char('[') => Msg::PrevHeading,
        KeyCode::Char('t') => Msg::ToggleOutline,
        KeyCode::Char('/') => Msg::StartSearch,
        KeyCode::Char('n') => Msg::SearchNext,
        KeyCode::Char('N') => Msg::SearchPrev,
        KeyCode::Char('f') => Msg::StartHints,
        KeyCode::Backspace => Msg::Back,
        KeyCode::Char('y') => Msg::Yank,
        KeyCode::Char('e') => Msg::Edit,
        KeyCode::Char('T') => Msg::CycleTheme,
        KeyCode::Char('?') => Msg::ToggleHelp,
        KeyCode::Esc => Msg::Cancel,
        KeyCode::Char('q' | 'Q') => Msg::Quit,
        _ => return None,
    })
}
