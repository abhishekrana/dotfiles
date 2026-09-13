//! The interactive reader: model, messages, update, and the event loop.

mod keys;
pub mod links;
mod position;
pub mod search;
pub mod selection;
pub mod trace;
mod watch;

use std::io::{Write as _, stdout};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{self, Receiver, Sender};
use std::thread;
use std::time::Duration;

use crossterm::event::{self, DisableMouseCapture, EnableMouseCapture, Event, MouseButton, MouseEventKind};
use crossterm::execute;
use tracing::{debug, info, warn};

use crate::buffer::Buffer;
use crate::doc::{self, Block, Document};
use crate::layout::{Layouter, Page};
use crate::style::Style;
use crate::theme::{self, Theme};
use links::Action;
use position::Positions;
use search::Match;
use selection::{Range, Spot};
use watch::FileWatch;

/// How long the input thread waits for a key before checking whether it should pause.
const INPUT_POLL: Duration = Duration::from_millis(100);

/// What wakes the event loop: the terminal, or the file on disk.
#[derive(Debug)]
pub enum Input {
    Term(Event),
    FileChanged,
    WatchError(String),
}

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
    /// Typed text for the search line.
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
    /// Re-read the file, keeping the top line on screen.
    Reload,
    /// Start or stop following the file on disk.
    ToggleWatch,
    /// Mouse press: anchor a selection at this screen cell.
    Press { col: u16, row: u16 },
    /// Mouse drag: extend the selection to this screen cell.
    DragTo { col: u16, row: u16 },
    /// Mouse release: copy the selection, or follow a link when the drag never moved.
    Release { col: u16, row: u16 },
    Quit,
}

/// What the reader is doing besides showing the page.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Mode {
    Read,
    Outline { selected: usize },
    Search { query: String },
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
    /// Follow the file on disk and reload on change.
    watching: bool,
    watch: Option<FileWatch>,
    /// Where the watcher and the input thread deliver; set by `run`.
    tx: Option<Sender<Input>>,
    positions: Option<Positions>,
    /// Record action edges in the shared trace log; only the running reader does.
    trace: bool,
    /// Set while an editor owns the terminal, so the input thread leaves the keys to it.
    input_paused: Arc<AtomicBool>,
    /// The mouse selection, while dragging and until the next press or Esc.
    selection: Option<Range>,
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
            watching: true,
            watch: None,
            tx: None,
            positions: None,
            trace: false,
            input_paused: Arc::new(AtomicBool::new(false)),
            selection: None,
        }
    }

    /// Whether to follow the file on disk; on unless asked otherwise.
    #[must_use]
    pub fn with_watch(mut self, on: bool) -> Self {
        self.watching = on;
        self
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
        self.selection = None;
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
            Msg::Input(c) => self.input(c),
            Msg::Backspace => {
                if let Mode::Search { query } = &mut self.mode {
                    query.pop();
                }
            }
            Msg::Select => self.select(),
            Msg::Up | Msg::Down => self.move_selection(msg == Msg::Down),
            Msg::Cancel => {
                if self.mode == Mode::Read {
                    self.clear_search();
                    self.selection = None;
                }
                self.mode = Mode::Read;
            }
            Msg::Back => self.back(),
            Msg::Yank => self.yank(),
            Msg::Reload => self.reload("reloaded"),
            Msg::ToggleWatch => self.toggle_watch(),
            Msg::Press { col, row } => self.press(col, row),
            Msg::DragTo { col, row } => self.drag_to(col, row),
            Msg::Release { col, row } => self.release(col, row),
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
        self.trace_edge("theme", &[("to", &next.id)]);
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
        if let Mode::Search { query } = &mut self.mode {
            query.push(c);
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

    /// The page cell under a pointer at a screen cell, clamped to the row's text.
    fn spot(&self, col: u16, row: u16) -> Option<Spot> {
        (self.mode == Mode::Read && usize::from(row) < self.body_rows).then(|| {
            let row = (self.scroll + usize::from(row)).min(self.page.lines.len().saturating_sub(1));
            let col = selection::column(&self.page, col).min(selection::row_width(&self.page, row));
            Spot { row, col }
        })
    }

    fn press(&mut self, col: u16, row: u16) {
        self.selection = self.spot(col, row).map(Range::new);
    }

    fn drag_to(&mut self, col: u16, row: u16) {
        if let (Some(cursor), Some(sel)) = (self.spot(col, row), self.selection.as_mut()) {
            sel.cursor = cursor;
        }
    }

    /// A drag copies what it covered; a press that never moved follows the link under it.
    fn release(&mut self, col: u16, row: u16) {
        self.drag_to(col, row);
        match self.selection {
            Some(sel) if !sel.is_empty() => self.copy_selection(sel),
            _ => {
                self.selection = None;
                self.follow_at(col, row);
            }
        }
    }

    fn follow_at(&mut self, col: u16, row: u16) {
        let Some(spot) = self.spot(col, row) else {
            return;
        };
        let Some(line) = self.page.lines.get(spot.row) else {
            return;
        };
        let mut at = spot.col;
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

    fn copy_selection(&mut self, sel: Range) {
        let text = sel.text(&self.page);
        if text.is_empty() {
            return;
        }
        let chars = text.chars().count();
        self.notice = Some(match copy(&text) {
            Ok(status) if status.success() => format!("copied {chars} characters"),
            Ok(status) => format!("clip exited with {status}"),
            Err(e) => format!("clip failed: {e}"),
        });
        info!(chars, notice = ?self.notice, "copy selection");
    }

    #[must_use]
    pub fn selection(&self) -> Option<Range> {
        self.selection
    }

    fn follow(&mut self, link: &str) {
        info!(link, "follow");
        self.trace_edge("follow", &[("link", link)]);
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
        self.remember_position();
        self.buffer = buffer;
        self.doc = doc::parse(&self.buffer);
        self.clear_search();
        self.scroll = 0;
        self.relayout(self.page.width);
        self.restore_position();
        self.apply_watch();
        if let Some(p) = self.buffer.path() {
            let shown = p.display().to_string();
            self.trace_edge("open", &[("path", &shown)]);
        }
    }

    fn toggle_watch(&mut self) {
        self.watching = !self.watching;
        self.notice = Some(if self.watching { "watching" } else { "watch off" }.to_owned());
        self.apply_watch();
    }

    /// Starts or stops the watcher to match `watching`, for the current file.
    fn apply_watch(&mut self) {
        self.watch = None;
        if !self.watching {
            return;
        }
        let (Some(path), Some(tx)) = (self.buffer.path(), &self.tx) else {
            return;
        };
        match FileWatch::start(path, tx.clone()) {
            Ok(w) => self.watch = Some(w),
            Err(e) => {
                warn!(error = %e, "watch");
                self.notice = Some(e.to_string());
                self.watching = false;
            }
        }
    }

    /// Re-reads the file and lays it out again, keeping the top source line on screen.
    fn reload(&mut self, why: &str) {
        let line = self.top_line();
        if let Err(e) = self.buffer.reload() {
            self.notice = Some(e.to_string());
            return;
        }
        self.doc = doc::parse(&self.buffer);
        self.relayout(self.page.width);
        self.scroll_to_line(line);
        self.notice = Some(why.to_owned());
        info!(line, why, "reload");
    }

    /// Scrolls so the first row laid out from source line `line` (1-based) is at the top.
    fn scroll_to_line(&mut self, line: usize) {
        let target = line.saturating_sub(1);
        let row = self
            .page
            .lines
            .iter()
            .position(|l| l.src.is_some_and(|s| self.buffer.byte_to_line(s.start) >= target));
        if let Some(row) = row {
            self.scroll = row;
        }
        self.clamp();
    }

    fn remember_position(&mut self) {
        let line = self.top_line();
        if let (Some(p), Some(pos)) = (self.buffer.path(), &mut self.positions) {
            pos.set(p, line);
        }
    }

    fn restore_position(&mut self) {
        let line = self.buffer.path().and_then(|p| self.positions.as_ref()?.get(p));
        if let Some(line) = line {
            self.scroll_to_line(line);
        }
    }

    fn trace_edge(&self, evt: &str, fields: &[(&str, &str)]) {
        if self.trace {
            trace::edge(evt, fields);
        }
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
        if let Some(sel) = self.selection.filter(|s| !s.is_empty()) {
            self.copy_selection(sel);
            return;
        }
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
        self.notice = Some(match copy(&code) {
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
        let (tx, rx) = mpsc::channel();
        self.tx = Some(tx.clone());
        self.trace = true;
        if let Some(dir) = crate::log::state_dir() {
            self.positions = Some(Positions::open(&dir));
        }
        let mut terminal = ratatui::init();
        execute!(stdout(), EnableMouseCapture)?;
        spawn_input_thread(tx, Arc::clone(&self.input_paused));
        info!(path = ?self.buffer.path(), "open");
        let size = terminal.size()?;
        self.resize(size.width, size.height);
        self.restore_position();
        self.apply_watch();
        if let Some(p) = self.buffer.path() {
            let shown = p.display().to_string();
            self.trace_edge("open", &[("path", &shown)]);
        }
        let result = self.event_loop(&mut terminal, &rx);
        self.remember_position();
        if let Some(p) = &mut self.positions
            && let Err(e) = p.save()
        {
            warn!(error = %e, "positions not saved");
        }
        execute!(stdout(), DisableMouseCapture)?;
        ratatui::restore();
        result
    }

    fn event_loop(&mut self, terminal: &mut ratatui::DefaultTerminal, rx: &Receiver<Input>) -> std::io::Result<()> {
        loop {
            let size = terminal.size()?;
            self.resize(size.width, size.height);
            terminal.draw(|frame| crate::ui::draw(frame, self))?;
            let input = rx.recv().map_err(|_| std::io::Error::other("input thread ended"))?;
            let msg = match input {
                Input::Term(Event::Key(key)) => keys::map(key, &self.mode),
                Input::Term(Event::Mouse(m)) => match m.kind {
                    MouseEventKind::ScrollDown => Some(Msg::ScrollLines(3)),
                    MouseEventKind::ScrollUp => Some(Msg::ScrollLines(-3)),
                    MouseEventKind::Down(MouseButton::Left) => Some(Msg::Press {
                        col: m.column,
                        row: m.row,
                    }),
                    MouseEventKind::Drag(MouseButton::Left) => Some(Msg::DragTo {
                        col: m.column,
                        row: m.row,
                    }),
                    MouseEventKind::Up(MouseButton::Left) => Some(Msg::Release {
                        col: m.column,
                        row: m.row,
                    }),
                    _ => None,
                },
                Input::Term(Event::Resize(w, h)) => {
                    debug!(w, h, "resize");
                    None
                }
                Input::Term(_) => None,
                Input::FileChanged => {
                    self.reload("reloaded");
                    None
                }
                Input::WatchError(e) => {
                    self.notice = Some(format!("watch: {e}"));
                    None
                }
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
        self.input_paused.store(true, Ordering::Release);
        // Let the input thread finish its current poll so the editor gets every key.
        thread::sleep(INPUT_POLL + Duration::from_millis(20));
        execute!(stdout(), DisableMouseCapture)?;
        ratatui::restore();
        let status = Command::new(&editor).arg(format!("+{line}")).arg(&path).status();
        *terminal = ratatui::init();
        execute!(stdout(), EnableMouseCapture)?;
        self.input_paused.store(false, Ordering::Release);
        match status {
            Ok(s) if s.success() => {}
            Ok(s) => self.notice = Some(format!("{editor} exited with {s}")),
            Err(e) => self.notice = Some(format!("cannot run {editor}: {e}")),
        }
        let keep = self.notice.take();
        self.reload("reloaded");
        if keep.is_some() {
            self.notice = keep;
        }
        Ok(())
    }
}

/// Forwards terminal events to the loop, sleeping while an editor owns the terminal.
fn spawn_input_thread(tx: Sender<Input>, paused: Arc<AtomicBool>) {
    thread::spawn(move || {
        loop {
            if paused.load(Ordering::Acquire) {
                thread::sleep(Duration::from_millis(20));
                continue;
            }
            match event::poll(INPUT_POLL) {
                Ok(true) if !paused.load(Ordering::Acquire) => match event::read() {
                    Ok(ev) => {
                        if tx.send(Input::Term(ev)).is_err() {
                            return;
                        }
                    }
                    Err(_) => return,
                },
                Ok(_) => {}
                Err(_) => return,
            }
        }
    });
}

/// Puts `text` on the clipboard through `clip`, the one clipboard path.
fn copy(text: &str) -> std::io::Result<std::process::ExitStatus> {
    let mut child = Command::new("clip")
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()?;
    if let Some(mut stdin) = child.stdin.take() {
        stdin.write_all(text.as_bytes())?;
    }
    child.wait()
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

    /// A press and release on one cell: what a click is once drags are selections.
    fn click(a: &mut App, col: u16, row: u16) {
        a.update(Msg::Press { col, row });
        a.update(Msg::Release { col, row });
    }

    #[test]
    fn a_click_on_a_link_follows_it_and_a_missing_note_says_so() {
        let text = "# Top\n\nSee [below](#end) and [[Missing note]].\n\ntext\n\ntext\n\n## End\n\nhere\n";
        let mut a = app_from(text, 5);
        let left = a.page().left;
        // Row 2 is the paragraph; "See " is four cells, so the link starts at column 4.
        click(&mut a, left + 5, 2);
        assert_eq!(a.section(), "End");
        a.update(Msg::Top);
        click(&mut a, left + 22, 2);
        assert_eq!(a.notice(), Some("note not found: Missing note"));
        click(&mut a, left, 2);
        assert!(a.notice().is_none(), "plain text is not a link");
    }

    #[test]
    fn a_drag_selects_the_cells_it_covered_instead_of_following_a_link() {
        let text = "# Top\n\nSee [below](#end) and more.\n\n## End\n\nhere\n";
        let mut a = app_from(text, 5);
        let left = a.page().left;
        a.update(Msg::Press { col: left, row: 2 });
        a.update(Msg::DragTo { col: left + 7, row: 2 });
        a.update(Msg::Release { col: left + 7, row: 2 });
        let sel = a.selection().expect("selection");
        assert_eq!(sel.row_span(2), Some((0, 7)));
        assert_eq!(sel.text(a.page()), "See bel");
        assert_eq!(a.section(), "Top", "a drag over a link does not follow it");
    }

    #[test]
    fn escape_and_a_relayout_drop_the_selection() {
        let mut a = app_from("# Top\n\nsome words here\n", 5);
        let left = a.page().left;
        a.update(Msg::Press { col: left, row: 2 });
        a.update(Msg::DragTo { col: left + 4, row: 2 });
        assert!(a.selection().is_some());
        a.update(Msg::Cancel);
        assert!(a.selection().is_none());
        a.update(Msg::Press { col: left, row: 2 });
        a.update(Msg::DragTo { col: left + 4, row: 2 });
        a.resize(40, 10);
        assert!(a.selection().is_none(), "rows move under a relayout");
    }

    #[test]
    fn reload_keeps_the_top_source_line_and_watch_toggles() {
        let mut a = app(60, 11);
        a.update(Msg::HalfPage(1));
        a.update(Msg::HalfPage(1));
        let line = a.top_line();
        a.update(Msg::Reload);
        assert_eq!((a.top_line(), a.notice()), (line, Some("reloaded")));
        a.update(Msg::ToggleWatch);
        assert_eq!(a.notice(), Some("watch off"));
        a.update(Msg::ToggleWatch);
        assert_eq!(a.notice(), Some("watching"));
        assert!(a.watch.is_none(), "a buffer without a file has nothing to watch");
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
