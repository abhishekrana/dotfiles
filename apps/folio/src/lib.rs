//! folio: a markdown reader for the terminal that reads like a page.
//!
//! Data flows one way: `buffer` (rope) -> `doc` (block tree with spans) -> `layout` (lines at a measure,
//! shaped by a `style`) -> `render` (ratatui or ANSI). `theme` maps palette roles to colours; `app` owns the loop.

pub mod app;
pub mod buffer;
pub mod doc;
pub mod highlight;
pub mod layout;
pub mod log;
pub mod render;
pub mod style;
pub mod theme;
pub mod ui;
