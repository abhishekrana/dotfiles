//! A style: one rule per element, loaded from TOML. Colours are palette roles.

mod load;

pub use load::{StyleError, load, user_styles_dir};

use serde::Deserialize;

use crate::theme::Role;

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Style {
    pub name: String,
    /// Reading column width: the whole pane, or a cell count the pane may force narrower.
    pub measure: Measure,
    /// Where the column sits when the pane is wider than the measure.
    pub align: Align,
    /// Outline rail on the left (layout switch; drawn from phase 5).
    pub rail: bool,
    pub h1: HeadingRule,
    pub h2: HeadingRule,
    pub h3: HeadingRule,
    pub h4: HeadingRule,
    pub h5: HeadingRule,
    pub h6: HeadingRule,
    pub paragraph: ParagraphRule,
    pub strong: InlineRule,
    pub emph: InlineRule,
    pub strike: InlineRule,
    pub link: InlineRule,
    pub wikilink: InlineRule,
    pub tag: InlineRule,
    pub code_span: CodeSpanRule,
    pub code: CodeRule,
    pub quote: QuoteRule,
    pub callout: CalloutRule,
    pub list: ListRule,
    pub task: TaskRule,
    pub table: TableRule,
    pub rule: HrRule,
    pub front_matter: FrontMatterRule,
    pub footnote: FootnoteRule,
    pub image: ImageRule,
}

impl Style {
    #[must_use]
    pub fn heading(&self, level: u8) -> &HeadingRule {
        match level {
            1 => &self.h1,
            2 => &self.h2,
            3 => &self.h3,
            4 => &self.h4,
            5 => &self.h5,
            _ => &self.h6,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(untagged, rename_all = "lowercase")]
pub enum Measure {
    Cells(u16),
    Full(Full),
}

/// The one word `measure` accepts besides a number.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Full {
    Full,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Align {
    Left,
    Center,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HeadingRule {
    pub fg: Role,
    pub bold: bool,
    pub italic: bool,
    pub rule: HeadingLine,
    pub above: u8,
    pub below: u8,
}

/// Where a heading's underline runs.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum HeadingLine {
    None,
    Words,
    Column,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ParagraphRule {
    pub fg: Role,
    pub below: u8,
}

/// Attributes an inline element adds to the text it wraps; unset fields inherit.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(deny_unknown_fields, default)]
pub struct InlineRule {
    pub fg: Option<Role>,
    pub bg: Option<Role>,
    pub bold: bool,
    pub italic: bool,
    pub underline: bool,
    pub strike: bool,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CodeSpanRule {
    pub fg: Role,
    pub bg: Option<Role>,
    /// Space cells on each side of the span.
    pub pad: u8,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CodeRule {
    pub bg: Option<Role>,
    pub pad: u8,
    pub label: CodeLabel,
    pub above: u8,
    pub below: u8,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum CodeLabel {
    None,
    Above,
    Right,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct QuoteRule {
    pub bar: String,
    pub bar_fg: Role,
    pub fg: Role,
    pub italic: bool,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CalloutRule {
    pub bar: String,
    pub bar_fg: Role,
    pub title_fg: Role,
    pub icon: bool,
    #[serde(default)]
    pub bg: Option<Role>,
    pub kinds: CalloutKinds,
}

/// Accent role per GitHub alert kind.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CalloutKinds {
    pub note: Role,
    pub tip: Role,
    pub important: Role,
    pub warning: Role,
    pub caution: Role,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ListRule {
    pub bullet: String,
    pub bullet_fg: Role,
    pub nested: String,
    pub indent: u8,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TaskRule {
    pub done: String,
    pub todo: String,
    pub done_fg: Role,
    pub todo_fg: Role,
    pub done_text: Role,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TableRule {
    pub lines: TableLines,
    pub header_bold: bool,
    pub header_rule: bool,
    #[serde(default)]
    pub zebra: Option<Role>,
    pub line_fg: Role,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TableLines {
    None,
    Rules,
    Box,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HrRule {
    pub glyph: String,
    pub fg: Role,
    pub width: HrWidth,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum HrWidth {
    /// Repeat the glyph across the measure.
    Column,
    /// Draw the glyph once, centred.
    Glyph,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FrontMatterRule {
    #[serde(rename = "as")]
    pub as_: FrontMatterAs,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FrontMatterAs {
    Hidden,
    Line,
    List,
    Table,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FootnoteRule {
    pub marker: FootnoteMarker,
    pub fg: Role,
    pub text_fg: Role,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FootnoteMarker {
    Superscript,
    Bracket,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ImageRule {
    pub placeholder: String,
    pub fg: Role,
}
