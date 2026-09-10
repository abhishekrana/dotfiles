//! The interactive reader: model, messages, update, and the event loop.

mod keys;

use std::io::stdout;

use crossterm::event::{self, DisableMouseCapture, EnableMouseCapture, Event, MouseEventKind};
use crossterm::execute;
use tracing::{debug, info};

use crate::buffer::Buffer;
use crate::doc::{self, Document};
use crate::layout::{Layouter, Page};
use crate::style::Style;
use crate::theme::Theme;

/// What a key or mouse event asks for.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Msg {
    ScrollLines(i32),
    HalfPage(i32),
    Top,
    Bottom,
    Quit,
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
        }
    }

    pub fn resize(&mut self, width: u16, height: u16) {
        self.body_rows = usize::from(height.saturating_sub(1));
        if self.page.width != width {
            self.page = self.layouter.layout(&self.doc, &self.style, width);
        }
        self.clamp();
    }

    pub fn update(&mut self, msg: Msg) {
        match msg {
            Msg::ScrollLines(n) => self.scroll = self.scroll.saturating_add_signed(n as isize),
            Msg::HalfPage(dir) => {
                let step = (self.body_rows / 2).max(1) as isize * dir as isize;
                self.scroll = self.scroll.saturating_add_signed(step);
            }
            Msg::Top => self.scroll = 0,
            Msg::Bottom => self.scroll = usize::MAX,
            Msg::Quit => {}
        }
        self.clamp();
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

    #[must_use]
    pub fn doc(&self) -> &Document {
        &self.doc
    }

    #[must_use]
    pub fn buffer(&self) -> &Buffer {
        &self.buffer
    }

    /// Percent of the document scrolled past, 100 when it all fits.
    #[must_use]
    pub fn percent(&self) -> u8 {
        let max = self.max_scroll();
        (self.scroll * 100).checked_div(max).map_or(100, |p| p as u8)
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
                Event::Key(key) => keys::map(key),
                Event::Mouse(m) => match m.kind {
                    MouseEventKind::ScrollDown => Some(Msg::ScrollLines(3)),
                    MouseEventKind::ScrollUp => Some(Msg::ScrollLines(-3)),
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
                Some(m) => self.update(m),
                None => {}
            }
        }
    }
}
