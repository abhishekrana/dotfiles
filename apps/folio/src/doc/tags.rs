//! `#tag` detection inside plain text runs.

use super::{Inline, Span};

/// Splits `#word` tokens out of a text run; a tag starts at the text start or after whitespace.
pub(super) fn split_tags(text: &str, span: Span) -> Vec<Inline> {
    let mut out = Vec::new();
    let mut text_start = 0;
    let mut i = 0;
    let bytes = text.as_bytes();
    while i < bytes.len() {
        let at_boundary = i == 0 || bytes[i - 1].is_ascii_whitespace();
        if bytes[i] == b'#' && at_boundary {
            let end = tag_end(text, i + 1);
            if end > i + 1 {
                if text_start < i {
                    out.push(Inline::Text(text[text_start..i].to_owned(), sub(span, text_start, i)));
                }
                out.push(Inline::Tag(text[i + 1..end].to_owned(), sub(span, i, end)));
                text_start = end;
                i = end;
                continue;
            }
        }
        i += 1;
    }
    if text_start < text.len() {
        out.push(Inline::Text(
            text[text_start..].to_owned(),
            sub(span, text_start, text.len()),
        ));
    }
    out
}

/// End of a tag body starting at `from`: letters, digits, `_`, `-`, `/`; must start with a letter or `_`.
fn tag_end(text: &str, from: usize) -> usize {
    let mut end = from;
    for (i, ch) in text[from..].char_indices() {
        let ok = if i == 0 {
            ch.is_alphabetic() || ch == '_'
        } else {
            ch.is_alphanumeric() || matches!(ch, '_' | '-' | '/')
        };
        if !ok {
            break;
        }
        end = from + i + ch.len_utf8();
    }
    end
}

fn sub(span: Span, start: usize, end: usize) -> Span {
    Span {
        start: span.start + start,
        end: (span.start + end).min(span.end.max(span.start + end)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tags_split_out_of_text() {
        let parts = split_tags("see #homelab and #a/b now", Span { start: 10, end: 35 });
        let tags: Vec<_> = parts
            .iter()
            .filter_map(|p| {
                if let Inline::Tag(t, _) = p {
                    Some(t.as_str())
                } else {
                    None
                }
            })
            .collect();
        assert_eq!(tags, vec!["homelab", "a/b"]);
        assert!(matches!(&parts[0], Inline::Text(t, s) if t == "see " && s.start == 10));
    }

    #[test]
    fn a_hash_inside_a_word_or_before_a_digit_is_not_a_tag() {
        assert!(
            split_tags("issue#12 and #42", Span::default())
                .iter()
                .all(|p| matches!(p, Inline::Text(..)))
        );
    }
}
