//! The interactive reader: model, messages, update, and the event loop.

mod keys;
pub mod links;
pub mod search;

use std::io::{Write as _, stdout};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

use crossterm::event::{self, DisableMouseCapture, EnableMouseCapture, Event, MouseButton, MouseEventKind};
use crossterm::execute;
use tracing::{debug, info, warn};

use crate::buffer::Buffer;
use crate::doc::{self, Block, Document};
use crate::layout::{Layouter, Page};
use crate::style::Style;
use crate::theme::{self, Theme};
use links::{Action, Target};
use search::Match;

/// What a key or mouse event asks for.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Msg {
    ScrollLines(i32),
    HalfPage(i32),
    Top,
    Bottom,
    NextHeading,
    PrevHeading,
    /// Next flavor in the palette's order, wrapping.
    CycleTheme,
    ToggleOutline,
    ToggleHelp,
    StartSearch,
    SearchNext,
    SearchPrev,
    StartHints,
    /// Typed text for search or a hint label.
    Input(char),
    Backspace,
    /// Confirm in an overlay or the search line.
    Select,
    /// Move in the outline.
    Up,
    Down,
    /// Leave an overlay, or clear the search in the reader.
    Cancel,
    /// Return to the document a link was followed from.
    Back,
    /// Copy the code block nearest the top of the screen.
    Yank,
    /// Open the file in the editor at the top line.
    Edit,
    Click {
        col: u16,
        row: u16,
    },
    Quit,
}

/// What the reader is doing besides showing the page.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Mode {
    Read,
    Outline {
        selected: usize,
    },
    Search {
        query: String,
    },
    Hints {
        typed: String,
        targets: Vec<Target>,
        labels: Vec<String>,
    },
    Help,
}

#[derive(Debug)]
pub struct App {
    buffer: Buffer,
    doc: Document,
    style: Style,
    theme: &'static Theme,
    layouter: Layouter,
    page: Page,
    /// First page row on screen.
    scroll: usize,
    /// Rows the page body has on screen.
    body_rows: usize,
    /// Page row where each heading starts, in document order.
    heading_rows: Vec<usize>,
    mode: Mode,
    query: String,
    matches: Vec<Match>,
    current_match: Option<usize>,
    /// Files followed from, for `Back`.
    history: Vec<PathBuf>,
    /// One-line message for the status row, cleared by the next key.
    notice: Option<String>,
}

impl App {
    #[must_use]
    pub fn new(buffer: Buffer, style: Style, theme: &'static Theme) -> Self {
        let doc = doc::parse(&buffer);
        Self {
            buffer,
            doc,
            style,
            theme,
            layouter: Layouter::new(theme),
            page: Page::default(),
            scroll: 0,
            body_rows: 0,
            heading_rows: Vec::new(),
            mode: Mode::Read,
            query: String::new(),
            matches: Vec::new(),
            current_match: None,
            history: Vec::new(),
            notice: None,
        }
    }

    pub fn resize(&mut self, width: u16, height: u16) {
        self.body_rows = usize::from(height.saturating_sub(1));
        if self.page.width != width {
            self.relayout(width);
        }
        self.clamp();
    }

    fn relayout(&mut self, width: u16) {
        self.page = self.layouter.layout(&self.doc, &self.style, width);
        self.heading_rows = self
            .doc
            .headings
            .iter()
            .map(|h| self.page.lines.iter().position(|l| l.block == h.block).unwrap_or(0))
            .collect();
        if !self.query.is_empty() {
            self.matches = search::find(&self.page, &self.query);
            self.current_match = self.current_match.filter(|i| *i < self.matches.len());
        }
    }

    pub fn update(&mut self, msg: Msg) {
        self.notice = None;
        match msg {
            Msg::ScrollLines(n) => self.scroll = self.scroll.saturating_add_signed(n as isize),
            Msg::HalfPage(dir) => {
                let step = (self.body_rows / 2).max(1) as isize * dir as isize;
                self.scroll = self.scroll.saturating_add_signed(step);
            }
            Msg::Top => self.scroll = 0,
            Msg::Bottom => self.scroll = usize::MAX,
            Msg::NextHeading => {
                if let Some(row) = self.heading_rows.iter().copied().find(|r| *r > self.scroll) {
                    self.scroll = row;
                }
            }
            Msg::PrevHeading => {
                if let Some(row) = self.heading_rows.iter().rev().copied().find(|r| *r < self.scroll) {
                    self.scroll = row;
                }
            }
            Msg::CycleTheme => self.cycle_theme(),
            Msg::ToggleOutline => self.toggle_outline(),
            Msg::ToggleHelp => {
                self.mode = if self.mode == Mode::Help {
                    Mode::Read
                } else {
                    Mode::Help
                };
            }
            Msg::StartSearch => {
                self.mode = Mode::Search {
                    query: self.query.clone(),
                }
            }
            Msg::SearchNext => self.step_match(1),
            Msg::SearchPrev => self.step_match(-1),
            Msg::StartHints => self.start_hints(),
            Msg::Input(c) => self.input(c),
            Msg::Backspace => match &mut self.mode {
                Mode::Search { query } => {
                    query.pop();
                }
                Mode::Hints { typed, .. } => {
                    typed.pop();
                }
                _ => {}
            },
            Msg::Select => self.select(),
            Msg::Up | Msg::Down => self.move_selection(msg == Msg::Down),
            Msg::Cancel => {
                if self.mode == Mode::Read {
                    self.clear_search();
                }
                self.mode = Mode::Read;
            }
            Msg::Back => self.back(),
            Msg::Yank => self.yank(),
            Msg::Click { col, row } => self.click(col, row),
            Msg::Edit | Msg::Quit => {}
        }
        self.clamp();
    }

    fn move_selection(&mut self, down: bool) {
        let n = self.doc.headings.len();
        if let Mode::Outline { selected } = &mut self.mode
            && n > 0
        {
            *selected = if down {
                (*selected + 1).min(n - 1)
            } else {
                selected.saturating_sub(1)
            };
        }
    }

    fn cycle_theme(&mut self) {
        let Ok(all) = theme::flavors() else { return };
        let at = all.iter().position(|t| t.id == self.theme.id).unwrap_or(0);
        let next = &all[(at + 1) % all.len()];
        info!(from = %self.theme.id, to = %next.id, "theme");
        self.theme = next;
        self.layouter.set_theme(next);
        self.relayout(self.page.width);
    }

    fn toggle_outline(&mut self) {
        self.mode = match self.mode {
            Mode::Outline { .. } => Mode::Read,
            _ if self.doc.headings.is_empty() => {
                self.notice = Some("no headings".to_owned());
                Mode::Read
            }
            _ => Mode::Outline {
                selected: self.current_heading().unwrap_or(0),
            },
        };
    }

    /// Index of the heading the top row sits under.
    fn current_heading(&self) -> Option<usize> {
        self.heading_rows.iter().rposition(|r| *r <= self.scroll)
    }

    fn input(&mut self, c: char) {
        match &mut self.mode {
            Mode::Search { query } => query.push(c),
            Mode::Hints { typed, targets, labels } => {
                typed.push(c);
                if let Some(i) = labels.iter().position(|l| l == typed) {
                    let target = targets[i].link.clone();
                    self.mode = Mode::Read;
                    self.follow(&target);
                } else if !labels.iter().any(|l| l.starts_with(typed.as_str())) {
                    self.mode = Mode::Read;
                    self.notice = Some("no such link".to_owned());
                }
            }
            _ => {}
        }
    }

    fn select(&mut self) {
        match &self.mode {
            Mode::Search { query } => {
                let query = query.clone();
                self.mode = Mode::Read;
                self.run_search(query);
            }
            Mode::Outline { selected } => {
                let row = self.heading_rows.get(*selected).copied();
                self.mode = Mode::Read;
                if let Some(row) = row {
                    self.scroll = row;
                }
            }
            _ => {}
        }
    }

    fn run_search(&mut self, query: String) {
        self.query = query;
        self.matches = search::find(&self.page, &self.query);
        if self.matches.is_empty() {
            self.current_match = None;
            if !self.query.is_empty() {
                self.notice = Some(format!("no match for {:?}", self.query));
            }
            return;
        }
        let first = self.matches.iter().position(|m| m.row >= self.scroll).unwrap_or(0);
        self.current_match = Some(first);
        self.scroll_to(self.matches[first].row);
        debug!(query = %self.query, matches = self.matches.len(), "search");
    }

    fn step_match(&mut self, dir: i32) {
        let n = self.matches.len();
        let Some(cur) = self.current_match.filter(|_| n > 0) else {
            if !self.query.is_empty() {
                let q = self.query.clone();
                self.run_search(q);
            }
            return;
        };
        let next = (cur as i64 + i64::from(dir)).rem_euclid(n as i64) as usize;
        self.current_match = Some(next);
        self.scroll_to(self.matches[next].row);
    }

    fn clear_search(&mut self) {
        self.query.clear();
        self.matches.clear();
        self.current_match = None;
    }

    /// Scrolls so `row` sits a third of the way down the screen.
    fn scroll_to(&mut self, row: usize) {
        self.scroll = row.saturating_sub(self.body_rows / 3);
        self.clamp();
    }

    fn start_hints(&mut self) {
        let targets = links::visible(&self.page, self.scroll, self.body_rows);
        if targets.is_empty() {
            self.notice = Some("no links on screen".to_owned());
            return;
        }
        let labels = links::labels(targets.len());
        self.mode = Mode::Hints {
            typed: String::new(),
            targets,
            labels,
        };
    }

    fn click(&mut self, col: u16, row: u16) {
        if self.mode != Mode::Read || usize::from(row) >= self.body_rows {
            return;
        }
        let Some(line) = self.page.lines.get(self.scroll + usize::from(row)) else {
            return;
        };
        let Some(mut at) = usize::from(col).checked_sub(usize::from(self.page.left)) else {
            return;
        };
        for s in &line.segments {
            let w = s.width();
            if at < w {
                if let Some(target) = &s.link {
                    let target = target.clone();
                    self.follow(&target);
                }
                return;
            }
            at -= w;
        }
    }

    fn follow(&mut self, link: &str) {
        info!(link, "follow");
        match links::classify(link) {
            Action::Anchor(id) => self.jump_to_anchor(&id),
            Action::Wiki { name, heading } => match links::resolve_wiki(&name, self.buffer.dir()) {
                Some(path) => self.open_note(&path, heading.as_deref()),
                None => self.notice = Some(format!("note not found: {name}")),
            },
            Action::Note { path, heading } => {
                let full = self.buffer.dir().map_or_else(|| path.clone(), |d| d.join(&path));
                self.open_note(&full, heading.as_deref());
            }
            Action::External(url) => match links::open_external(&url) {
                Ok(()) => self.notice = Some(format!("opened {url}")),
                Err(e) => {
                    warn!(url, error = %e, "open failed");
                    self.notice = Some(format!("cannot open {url}: {e}"));
                }
            },
        }
    }

    fn jump_to_anchor(&mut self, id: &str) {
        match self.doc.headings.iter().position(|h| h.id == id) {
            Some(i) => self.scroll = self.heading_rows[i],
            None => self.notice = Some(format!("no heading #{id}")),
        }
    }

    fn open_note(&mut self, path: &Path, heading: Option<&str>) {
        let previous = self.buffer.path().map(Path::to_path_buf);
        match Buffer::from_path(path) {
            Ok(buffer) => {
                if let Some(p) = previous {
                    self.history.push(p);
                }
                self.load(buffer);
                if let Some(h) = heading {
                    let id = h.to_lowercase().replace(' ', "-");
                    self.jump_to_anchor(&id);
                }
            }
            Err(e) => {
                warn!(error = %e, "open note");
                self.notice = Some(e.to_string());
            }
        }
    }

    fn load(&mut self, buffer: Buffer) {
        info!(path = ?buffer.path(), bytes = buffer.len_bytes(), "load");
        self.buffer = buffer;
        self.doc = doc::parse(&self.buffer);
        self.clear_search();
        self.scroll = 0;
        self.relayout(self.page.width);
    }

    fn back(&mut self) {
        let Some(path) = self.history.pop() else {
            self.notice = Some("nothing to go back to".to_owned());
            return;
        };
        match Buffer::from_path(&path) {
            Ok(buffer) => self.load(buffer),
            Err(e) => self.notice = Some(e.to_string()),
        }
    }

    /// Copies the first code block on screen through `clip`.
    fn yank(&mut self) {
        let code = self
            .page
            .lines
            .iter()
            .skip(self.scroll)
            .take(self.body_rows)
            .map(|l| l.block)
            .find_map(|i| match self.doc.blocks.get(i) {
                Some(Block::Code { text, .. }) => Some(text.clone()),
                _ => None,
            });
        let Some(code) = code else {
            self.notice = Some("no code block on screen".to_owned());
            return;
        };
        let lines = code.lines().count();
        let result = Command::new("clip")
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .spawn()
            .and_then(|mut child| {
                if let Some(mut stdin) = child.stdin.take() {
                    stdin.write_all(code.as_bytes())?;
                }
                child.wait()
            });
        self.notice = Some(match result {
            Ok(status) if status.success() => format!("copied {lines} lines"),
            Ok(status) => format!("clip exited with {status}"),
            Err(e) => format!("clip failed: {e}"),
        });
        info!(lines, notice = ?self.notice, "yank");
    }

    /// Line number (1-based) of the source at the top of the screen.
    fn top_line(&self) -> usize {
        self.page
            .lines
            .iter()
            .skip(self.scroll)
            .find_map(|l| l.src)
            .map_or(1, |span| self.buffer.byte_to_line(span.start) + 1)
    }

    fn clamp(&mut self) {
        self.scroll = self.scroll.min(self.max_scroll());
    }

    #[must_use]
    pub fn max_scroll(&self) -> usize {
        self.page.lines.len().saturating_sub(self.body_rows)
    }

    #[must_use]
    pub fn scroll(&self) -> usize {
        self.scroll
    }

    #[must_use]
    pub fn body_rows(&self) -> usize {
        self.body_rows
    }

    #[must_use]
    pub fn page(&self) -> &Page {
        &self.page
    }

    #[must_use]
    pub fn theme(&self) -> &Theme {
        self.theme
    }

    /// The theme's id when it is not the palette's default, for the status line.
    #[must_use]
    pub fn theme_label(&self) -> Option<&str> {
        let default = Theme::default_theme().ok()?;
        (self.theme.id != default.id).then_some(self.theme.id.as_str())
    }

    #[must_use]
    pub fn doc(&self) -> &Document {
        &self.doc
    }

    #[must_use]
    pub fn buffer(&self) -> &Buffer {
        &self.buffer
    }

    #[must_use]
    pub fn mode(&self) -> &Mode {
        &self.mode
    }

    #[must_use]
    pub fn matches(&self) -> &[Match] {
        &self.matches
    }

    #[must_use]
    pub fn current_match(&self) -> Option<usize> {
        self.current_match
    }

    #[must_use]
    pub fn notice(&self) -> Option<&str> {
        self.notice.as_deref()
    }

    /// Percent of the document scrolled past, 100 when it all fits.
    #[must_use]
    pub fn percent(&self) -> u8 {
        let max = self.max_scroll();
        (self.scroll * 100).checked_div(max).map_or(100, |p| p as u8)
    }

    /// Text of the heading the top row sits under.
    #[must_use]
    pub fn section(&self) -> &str {
        self.current_heading()
            .map_or("", |i| self.doc.headings[i].text.as_str())
    }

    /// Runs the terminal loop until quit.
    pub fn run(mut self) -> std::io::Result<()> {
        let mut terminal = ratatui::init();
        execute!(stdout(), EnableMouseCapture)?;
        info!(path = ?self.buffer.path(), "open");
        let result = self.event_loop(&mut terminal);
        execute!(stdout(), DisableMouseCapture)?;
        ratatui::restore();
        result
    }

    fn event_loop(&mut self, terminal: &mut ratatui::DefaultTerminal) -> std::io::Result<()> {
        loop {
            let size = terminal.size()?;
            self.resize(size.width, size.height);
            terminal.draw(|frame| crate::ui::draw(frame, self))?;
            let msg = match event::read()? {
                Event::Key(key) => keys::map(key, &self.mode),
                Event::Mouse(m) => match m.kind {
                    MouseEventKind::ScrollDown => Some(Msg::ScrollLines(3)),
                    MouseEventKind::ScrollUp => Some(Msg::ScrollLines(-3)),
                    MouseEventKind::Up(MouseButton::Left) => Some(Msg::Click {
                        col: m.column,
                        row: m.row,
                    }),
                    _ => None,
                },
                Event::Resize(w, h) => {
                    debug!(w, h, "resize");
                    None
                }
                _ => None,
            };
            match msg {
                Some(Msg::Quit) => return Ok(()),
                Some(Msg::Edit) => self.edit(terminal)?,
                Some(m) => self.update(m),
                None => {}
            }
        }
    }

    /// Suspends the reader, opens the file in the editor at the top line, then reloads it.
    fn edit(&mut self, terminal: &mut ratatui::DefaultTerminal) -> std::io::Result<()> {
        self.notice = None;
        let Some(path) = self.buffer.path().map(Path::to_path_buf) else {
            self.notice = Some("no file to edit".to_owned());
            return Ok(());
        };
        let editor = std::env::var("VISUAL")
            .or_else(|_| std::env::var("EDITOR"))
            .unwrap_or_else(|_| "vi".to_owned());
        let line = self.top_line();
        info!(%editor, line, "edit");
        execute!(stdout(), DisableMouseCapture)?;
        ratatui::restore();
        let status = Command::new(&editor).arg(format!("+{line}")).arg(&path).status();
        *terminal = ratatui::init();
        execute!(stdout(), EnableMouseCapture)?;
        match status {
            Ok(s) if s.success() => {}
            Ok(s) => self.notice = Some(format!("{editor} exited with {s}")),
            Err(e) => self.notice = Some(format!("cannot run {editor}: {e}")),
        }
        if let Err(e) = self.buffer.reload() {
            self.notice = Some(e.to_string());
            return Ok(());
        }
        let scroll = self.scroll;
        self.doc = doc::parse(&self.buffer);
        self.relayout(self.page.width);
        self.scroll = scroll;
        self.clamp();
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn app_from(text: &str, height: u16) -> App {
        let style = crate::style::load("github", None).expect("style");
        let theme = Theme::default_theme().expect("theme");
        let mut app = App::new(Buffer::from_text(text), style, theme);
        app.resize(100, height);
        app
    }

    fn app(lines: usize, height: u16) -> App {
        let text = (0..lines).map(|i| format!("Line {i}\n")).collect::<Vec<_>>().join("\n");
        app_from(&text, height)
    }

    #[test]
    fn a_short_document_has_nothing_to_scroll() {
        let mut a = app(3, 20);
        assert_eq!(a.max_scroll(), 0);
        a.update(Msg::ScrollLines(5));
        assert_eq!((a.scroll(), a.percent()), (0, 100));
    }

    #[test]
    fn scrolling_clamps_at_both_ends_and_pages_by_half_the_body() {
        let mut a = app(50, 11);
        assert_eq!(a.body_rows(), 10);
        a.update(Msg::ScrollLines(-3));
        assert_eq!(a.scroll(), 0);
        a.update(Msg::HalfPage(1));
        assert_eq!(a.scroll(), 5);
        a.update(Msg::Bottom);
        assert_eq!((a.scroll(), a.percent()), (a.max_scroll(), 100));
        a.update(Msg::ScrollLines(1));
        assert_eq!(a.scroll(), a.max_scroll());
        a.update(Msg::Top);
        assert_eq!((a.scroll(), a.percent()), (0, 0));
    }

    #[test]
    fn cycling_the_theme_walks_the_palette_and_wraps() {
        let mut a = app(3, 20);
        let n = theme::flavors().expect("palette").len();
        assert!(a.theme_label().is_none(), "starts on the default");
        a.update(Msg::CycleTheme);
        assert_eq!(a.theme_label(), Some("solarized-dark"));
        for _ in 1..n {
            a.update(Msg::CycleTheme);
        }
        assert!(a.theme_label().is_none(), "wrapped back to the default");
    }

    #[test]
    fn a_resize_relayouts_and_keeps_the_scroll_in_range() {
        let mut a = app(50, 11);
        a.update(Msg::Bottom);
        let at_bottom = a.scroll();
        a.resize(100, 60);
        assert!(a.scroll() <= a.max_scroll() && a.scroll() < at_bottom);
    }

    const SECTIONS: &str = "# One\n\ntext\n\ntext\n\n## Two\n\ntext\n\n## Three\n\ntext\n\ntext\n\ntext\n";

    #[test]
    fn heading_jumps_walk_the_outline_both_ways() {
        let mut a = app_from(SECTIONS, 6);
        assert_eq!(a.section(), "One");
        a.update(Msg::NextHeading);
        assert_eq!(a.section(), "Two");
        a.update(Msg::NextHeading);
        assert_eq!(a.section(), "Three");
        a.update(Msg::NextHeading);
        assert_eq!(a.section(), "Three", "the last heading is the end");
        a.update(Msg::PrevHeading);
        assert_eq!(a.section(), "Two");
    }

    #[test]
    fn the_outline_opens_on_the_current_section_and_jumps_on_select() {
        let mut a = app_from(SECTIONS, 6);
        a.update(Msg::NextHeading);
        a.update(Msg::ToggleOutline);
        assert_eq!(a.mode(), &Mode::Outline { selected: 1 });
        a.update(Msg::Down);
        a.update(Msg::Select);
        assert_eq!((a.mode(), a.section()), (&Mode::Read, "Three"));
    }

    #[test]
    fn search_moves_between_matches_and_escape_clears_it() {
        let mut a = app_from("alpha\n\nbeta ALPHA\n\ngamma\n\nalphabet\n", 4);
        a.update(Msg::StartSearch);
        for c in "alpha".chars() {
            a.update(Msg::Input(c));
        }
        a.update(Msg::Select);
        assert_eq!(a.matches().len(), 3);
        assert_eq!(a.current_match(), Some(0));
        a.update(Msg::SearchNext);
        assert_eq!(a.current_match(), Some(1));
        a.update(Msg::SearchPrev);
        a.update(Msg::SearchPrev);
        assert_eq!(a.current_match(), Some(2), "wraps backwards");
        a.update(Msg::Cancel);
        assert!(a.matches().is_empty());
    }

    #[test]
    fn a_missing_search_leaves_a_notice() {
        let mut a = app_from("text\n", 4);
        a.update(Msg::StartSearch);
        a.update(Msg::Input('z'));
        a.update(Msg::Select);
        assert_eq!(a.notice(), Some("no match for \"z\""));
    }

    #[test]
    fn hints_label_visible_links_and_an_anchor_jumps() {
        let text = "# Top\n\nSee [below](#end) and [[Missing note]].\n\ntext\n\ntext\n\n## End\n\nhere\n";
        let mut a = app_from(text, 5);
        a.update(Msg::StartHints);
        let Mode::Hints { labels, targets, .. } = a.mode().clone() else {
            panic!("hints mode")
        };
        assert_eq!(labels, vec!["a", "s"]);
        assert_eq!(targets.len(), 2);
        a.update(Msg::Input('a'));
        assert_eq!((a.mode(), a.section()), (&Mode::Read, "End"));
        a.update(Msg::Top);
        a.update(Msg::StartHints);
        a.update(Msg::Input('s'));
        assert_eq!(a.notice(), Some("note not found: Missing note"));
        a.update(Msg::StartHints);
        a.update(Msg::Input('z'));
        assert_eq!(a.notice(), Some("no such link"));
    }

    #[test]
    fn yank_without_a_code_block_says_so() {
        let mut a = app_from("text\n", 4);
        a.update(Msg::Yank);
        assert_eq!(a.notice(), Some("no code block on screen"));
        a.update(Msg::Back);
        assert_eq!(a.notice(), Some("nothing to go back to"));
    }
}
