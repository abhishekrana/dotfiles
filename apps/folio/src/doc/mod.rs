//! The document model: a block tree where every node carries its byte span in the buffer.

mod parse;
mod tags;

pub use parse::parse;

/// Byte range in the buffer, end exclusive.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
pub struct Span {
    pub start: usize,
    pub end: usize,
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum Block {
    FrontMatter {
        fields: Vec<(String, String)>,
        span: Span,
    },
    Heading {
        level: u8,
        inlines: Vec<Inline>,
        id: String,
        span: Span,
    },
    Paragraph {
        inlines: Vec<Inline>,
        span: Span,
    },
    List {
        ordered: bool,
        start: usize,
        /// No blank rows between items or inside them (`CommonMark` tightness).
        tight: bool,
        items: Vec<ListItem>,
        span: Span,
    },
    Quote {
        blocks: Vec<Block>,
        span: Span,
    },
    Callout {
        kind: CalloutKind,
        title: Option<String>,
        blocks: Vec<Block>,
        span: Span,
    },
    Code {
        lang: Option<String>,
        text: String,
        span: Span,
    },
    Table {
        align: Vec<Align>,
        head: Vec<Vec<Inline>>,
        rows: Vec<Vec<Vec<Inline>>>,
        span: Span,
    },
    Rule {
        span: Span,
    },
    Footnote {
        label: String,
        blocks: Vec<Block>,
        span: Span,
    },
    Image {
        alt: String,
        src: String,
        span: Span,
    },
}

impl Block {
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Block::FrontMatter { span, .. }
            | Block::Heading { span, .. }
            | Block::Paragraph { span, .. }
            | Block::List { span, .. }
            | Block::Quote { span, .. }
            | Block::Callout { span, .. }
            | Block::Code { span, .. }
            | Block::Table { span, .. }
            | Block::Rule { span }
            | Block::Footnote { span, .. }
            | Block::Image { span, .. } => *span,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct ListItem {
    /// `Some(done)` for a task item.
    pub task: Option<bool>,
    pub blocks: Vec<Block>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum CalloutKind {
    Note,
    Tip,
    Important,
    Warning,
    Caution,
}

impl CalloutKind {
    #[must_use]
    pub fn title(self) -> &'static str {
        match self {
            CalloutKind::Note => "Note",
            CalloutKind::Tip => "Tip",
            CalloutKind::Important => "Important",
            CalloutKind::Warning => "Warning",
            CalloutKind::Caution => "Caution",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Align {
    Left,
    Center,
    Right,
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum Inline {
    Text(String, Span),
    Code(String, Span),
    Strong(Vec<Inline>, Span),
    Emph(Vec<Inline>, Span),
    Strike(Vec<Inline>, Span),
    Link {
        text: Vec<Inline>,
        href: String,
        span: Span,
    },
    WikiLink {
        target: String,
        alias: Option<String>,
        span: Span,
    },
    Tag(String, Span),
    FootnoteRef(String, Span),
    Image {
        alt: String,
        src: String,
        span: Span,
    },
    SoftBreak,
    HardBreak,
}

/// A heading, for the outline and the status line.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct HeadingRef {
    pub block: usize,
    pub level: u8,
    pub text: String,
    pub id: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Document {
    pub blocks: Vec<Block>,
    pub headings: Vec<HeadingRef>,
}

/// Plain text of an inline run, for slugs, widths and the outline.
#[must_use]
pub fn plain_text(inlines: &[Inline]) -> String {
    let mut out = String::new();
    push_plain(inlines, &mut out);
    out
}

fn push_plain(inlines: &[Inline], out: &mut String) {
    for inline in inlines {
        match inline {
            Inline::Text(s, _) | Inline::Code(s, _) => out.push_str(s),
            Inline::Strong(inner, _) | Inline::Emph(inner, _) | Inline::Strike(inner, _) => push_plain(inner, out),
            Inline::Link { text, .. } => push_plain(text, out),
            Inline::WikiLink { target, alias, .. } => out.push_str(alias.as_deref().unwrap_or(target)),
            Inline::Tag(s, _) => {
                out.push('#');
                out.push_str(s);
            }
            Inline::FootnoteRef(..) => {}
            Inline::Image { alt, .. } => out.push_str(alt),
            Inline::SoftBreak | Inline::HardBreak => out.push(' '),
        }
    }
}
