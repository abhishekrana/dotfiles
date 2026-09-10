//! comrak AST -> `Document`. Spans come from comrak's line/column positions via the buffer's line table.

use std::collections::HashMap;

use comrak::nodes::{AlertType, AstNode, ListType, NodeValue, Sourcepos, TableAlignment};
use comrak::{Arena, Options};
use tracing::debug;

use super::tags::split_tags;
use super::{Align, Block, CalloutKind, Document, HeadingRef, Inline, ListItem, Span, plain_text};
use crate::buffer::Buffer;

/// Parses the buffer's text into a document.
#[must_use]
pub fn parse(buffer: &Buffer) -> Document {
    let text = buffer.text();
    let arena = Arena::new();
    let root = comrak::parse_document(&arena, &text, &options());
    let mut cx = Context {
        buffer,
        text_len: text.len(),
        footnotes: Vec::new(),
        slugs: HashMap::new(),
    };
    let mut blocks = cx.blocks(root);
    blocks.append(&mut cx.footnotes);
    let headings = blocks
        .iter()
        .enumerate()
        .filter_map(|(i, b)| match b {
            Block::Heading { level, inlines, id, .. } => Some(HeadingRef {
                block: i,
                level: *level,
                text: plain_text(inlines),
                id: id.clone(),
            }),
            _ => None,
        })
        .collect();
    debug!(blocks = blocks.len(), "parsed");
    Document { blocks, headings }
}

fn options() -> Options<'static> {
    let mut o = Options::default();
    o.extension.table = true;
    o.extension.tasklist = true;
    o.extension.strikethrough = true;
    o.extension.autolink = true;
    o.extension.footnotes = true;
    o.extension.front_matter_delimiter = Some("---".to_owned());
    o.extension.alerts = true;
    o.extension.wikilinks_title_after_pipe = true;
    o
}

struct Context<'b> {
    buffer: &'b Buffer,
    text_len: usize,
    footnotes: Vec<Block>,
    slugs: HashMap<String, usize>,
}

impl Context<'_> {
    fn span(&self, pos: Sourcepos) -> Span {
        let start = self.byte_at(pos.start.line, pos.start.column.saturating_sub(1));
        let end = self.byte_at(pos.end.line, pos.end.column);
        Span {
            start: start.min(self.text_len),
            end: end.min(self.text_len).max(start),
        }
    }

    fn byte_at(&self, line: usize, column: usize) -> usize {
        self.buffer
            .line_to_byte(line.saturating_sub(1))
            .map_or(self.text_len, |b| b + column)
    }

    fn blocks<'a>(&mut self, parent: &'a AstNode<'a>) -> Vec<Block> {
        let mut out = Vec::new();
        for node in parent.children() {
            if let Some(block) = self.block(node) {
                out.push(block);
            }
        }
        out
    }

    fn block<'a>(&mut self, node: &'a AstNode<'a>) -> Option<Block> {
        let data = node.data.borrow();
        let span = self.span(data.sourcepos);
        let block = match &data.value {
            NodeValue::FrontMatter(raw) => Block::FrontMatter {
                fields: front_matter_fields(raw),
                span,
            },
            NodeValue::Heading(h) => {
                let inlines = self.inlines(node, false);
                let id = self.slug(&plain_text(&inlines));
                Block::Heading {
                    level: h.level,
                    inlines,
                    id,
                    span,
                }
            }
            NodeValue::Paragraph => {
                let inlines = self.inlines(node, true);
                match inlines.as_slice() {
                    [Inline::Image { alt, src, .. }] => Block::Image {
                        alt: alt.clone(),
                        src: src.clone(),
                        span,
                    },
                    _ => Block::Paragraph { inlines, span },
                }
            }
            NodeValue::List(list) => Block::List {
                ordered: list.list_type == ListType::Ordered,
                start: list.start,
                tight: list.tight,
                items: node.children().map(|item| self.list_item(item)).collect(),
                span,
            },
            NodeValue::BlockQuote | NodeValue::MultilineBlockQuote(_) => Block::Quote {
                blocks: self.blocks(node),
                span,
            },
            NodeValue::Alert(alert) => Block::Callout {
                kind: callout_kind(alert.alert_type),
                title: alert.title.clone(),
                blocks: self.blocks(node),
                span,
            },
            NodeValue::CodeBlock(code) => Block::Code {
                lang: code.info.split_whitespace().next().map(str::to_owned),
                text: code.literal.clone(),
                span,
            },
            NodeValue::HtmlBlock(html) => Block::Code {
                lang: Some("html".to_owned()),
                text: html.literal.clone(),
                span,
            },
            NodeValue::ThematicBreak => Block::Rule { span },
            NodeValue::Table(table) => self.table(node, &table.alignments, span),
            NodeValue::FootnoteDefinition(def) => {
                let blocks = self.blocks(node);
                self.footnotes.push(Block::Footnote {
                    label: def.name.clone(),
                    blocks,
                    span,
                });
                return None;
            }
            other => {
                debug!(node = ?std::mem::discriminant(other), "skipping unsupported block");
                return None;
            }
        };
        Some(block)
    }

    fn list_item<'a>(&mut self, node: &'a AstNode<'a>) -> ListItem {
        let task = match &node.data.borrow().value {
            NodeValue::TaskItem(t) => Some(t.symbol.is_some()),
            _ => None,
        };
        ListItem {
            task,
            blocks: self.blocks(node),
        }
    }

    fn table<'a>(&mut self, node: &'a AstNode<'a>, alignments: &[TableAlignment], span: Span) -> Block {
        let mut head = Vec::new();
        let mut rows = Vec::new();
        for row in node.children() {
            let is_header = matches!(row.data.borrow().value, NodeValue::TableRow(true));
            let cells: Vec<Vec<Inline>> = row.children().map(|cell| self.inlines(cell, false)).collect();
            if is_header {
                head = cells;
            } else {
                rows.push(cells);
            }
        }
        let align = alignments
            .iter()
            .map(|a| match a {
                TableAlignment::Center => Align::Center,
                TableAlignment::Right => Align::Right,
                TableAlignment::None | TableAlignment::Left => Align::Left,
            })
            .collect();
        Block::Table {
            align,
            head,
            rows,
            span,
        }
    }

    fn inlines<'a>(&mut self, parent: &'a AstNode<'a>, tags: bool) -> Vec<Inline> {
        let mut out = Vec::new();
        for node in parent.children() {
            self.inline(node, tags, &mut out);
        }
        out
    }

    fn inline<'a>(&mut self, node: &'a AstNode<'a>, tags: bool, out: &mut Vec<Inline>) {
        let data = node.data.borrow();
        let span = self.span(data.sourcepos);
        match &data.value {
            NodeValue::Text(text) => {
                if tags && text.contains('#') {
                    out.extend(split_tags(text, span));
                } else {
                    out.push(Inline::Text(text.to_string(), span));
                }
            }
            NodeValue::Code(code) => out.push(Inline::Code(code.literal.clone(), span)),
            NodeValue::Emph => out.push(Inline::Emph(self.inlines(node, tags), span)),
            NodeValue::Strong => out.push(Inline::Strong(self.inlines(node, tags), span)),
            NodeValue::Strikethrough => out.push(Inline::Strike(self.inlines(node, tags), span)),
            NodeValue::Link(link) => {
                out.push(Inline::Link {
                    text: self.inlines(node, false),
                    href: link.url.clone(),
                    span,
                });
            }
            NodeValue::Image(link) => {
                let alt = plain_text(&self.inlines(node, false));
                out.push(Inline::Image {
                    alt,
                    src: link.url.clone(),
                    span,
                });
            }
            NodeValue::WikiLink(link) => {
                let title = plain_text(&self.inlines(node, false));
                let alias = (title != link.url).then_some(title);
                out.push(Inline::WikiLink {
                    target: link.url.clone(),
                    alias,
                    span,
                });
            }
            NodeValue::FootnoteReference(fr) => out.push(Inline::FootnoteRef(fr.name.clone(), span)),
            NodeValue::SoftBreak => out.push(Inline::SoftBreak),
            NodeValue::LineBreak => out.push(Inline::HardBreak),
            NodeValue::HtmlInline(html) | NodeValue::Raw(html) => out.push(Inline::Text(html.clone(), span)),
            NodeValue::Escaped | NodeValue::Underline | NodeValue::Highlight | NodeValue::Insert => {
                for child in node.children() {
                    self.inline(child, tags, out);
                }
            }
            other => debug!(node = ?std::mem::discriminant(other), "skipping unsupported inline"),
        }
    }

    /// GitHub-style heading id, unique within the document.
    fn slug(&mut self, text: &str) -> String {
        let mut slug = String::with_capacity(text.len());
        for ch in text.chars() {
            if ch.is_alphanumeric() {
                slug.extend(ch.to_lowercase());
            } else if (ch == ' ' || ch == '-') && !slug.ends_with('-') {
                slug.push('-');
            }
        }
        let slug = slug.trim_matches('-').to_owned();
        let n = self.slugs.entry(slug.clone()).or_insert(0);
        let unique = if *n == 0 { slug.clone() } else { format!("{slug}-{n}") };
        *n += 1;
        unique
    }
}

fn callout_kind(kind: AlertType) -> CalloutKind {
    match kind {
        AlertType::Note => CalloutKind::Note,
        AlertType::Tip => CalloutKind::Tip,
        AlertType::Important => CalloutKind::Important,
        AlertType::Warning => CalloutKind::Warning,
        AlertType::Caution => CalloutKind::Caution,
    }
}

/// `key: value` lines of the front matter; anything else is kept as a value-less line.
fn front_matter_fields(raw: &str) -> Vec<(String, String)> {
    raw.lines()
        .map(str::trim)
        .filter(|l| !l.is_empty() && !l.starts_with("---"))
        .map(|l| match l.split_once(':') {
            Some((k, v)) => (k.trim().to_owned(), v.trim().to_owned()),
            None => (l.to_owned(), String::new()),
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn doc(text: &str) -> Document {
        parse(&Buffer::from_text(text))
    }

    #[test]
    fn heading_spans_cover_the_source_line() {
        let text = "# Title\n\nBody.\n";
        let d = doc(text);
        let Block::Heading { span, id, .. } = &d.blocks[0] else {
            panic!("heading")
        };
        assert_eq!(&text[span.start..span.end], "# Title");
        assert_eq!(id, "title");
        assert_eq!(d.headings.len(), 1);
    }

    #[test]
    fn tasks_callouts_wikilinks_and_footnotes_parse() {
        let d =
            doc("- [x] done\n- [ ] todo\n\n> [!TIP]\n> hint\n\nSee [[Note|the note]] and #tag[^1].\n\n[^1]: foot\n");
        let Block::List { items, .. } = &d.blocks[0] else {
            panic!("list")
        };
        assert_eq!(
            items.iter().map(|i| i.task).collect::<Vec<_>>(),
            vec![Some(true), Some(false)]
        );
        assert!(matches!(
            &d.blocks[1],
            Block::Callout {
                kind: CalloutKind::Tip,
                ..
            }
        ));
        let Block::Paragraph { inlines, .. } = &d.blocks[2] else {
            panic!("paragraph")
        };
        assert!(inlines.iter().any(
            |i| matches!(i, Inline::WikiLink { target, alias: Some(a), .. } if target == "Note" && a == "the note")
        ));
        assert!(inlines.iter().any(|i| matches!(i, Inline::Tag(t, _) if t == "tag")));
        assert!(matches!(d.blocks.last(), Some(Block::Footnote { label, .. }) if label == "1"));
    }

    #[test]
    fn front_matter_becomes_fields() {
        let d = doc("---\ntype: guide\ntags: [a, b]\n---\n\n# T\n");
        let Block::FrontMatter { fields, .. } = &d.blocks[0] else {
            panic!("front matter")
        };
        assert_eq!(
            fields,
            &vec![
                ("type".to_owned(), "guide".to_owned()),
                ("tags".to_owned(), "[a, b]".to_owned())
            ]
        );
    }

    #[test]
    fn duplicate_headings_get_numbered_ids() {
        let d = doc("# A\n\n# A\n");
        assert_eq!(d.headings[1].id, "a-1");
    }
}
