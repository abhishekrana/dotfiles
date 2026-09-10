//! In-document search over the laid-out rows: case-insensitive, matches as column ranges.

use unicode_width::UnicodeWidthStr;

use crate::layout::Page;

/// One hit: a row of the page and the cell columns it covers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Match {
    pub row: usize,
    pub start: usize,
    pub end: usize,
}

/// Every non-overlapping occurrence of `query`, ignoring case, in page order.
#[must_use]
pub fn find(page: &Page, query: &str) -> Vec<Match> {
    let needle: Vec<char> = query.chars().filter_map(|c| c.to_lowercase().next()).collect();
    if needle.is_empty() {
        return Vec::new();
    }
    let mut out = Vec::new();
    for (row, line) in page.lines.iter().enumerate() {
        let text: String = line.segments.iter().map(|s| s.text.as_str()).collect();
        let chars: Vec<(usize, char)> = text.char_indices().collect();
        let lowered: Vec<char> = chars.iter().filter_map(|(_, c)| c.to_lowercase().next()).collect();
        let mut i = 0;
        while i + needle.len() <= lowered.len() {
            if lowered[i..i + needle.len()] == needle[..] {
                let start_byte = chars[i].0;
                let end_byte = chars.get(i + needle.len()).map_or(text.len(), |(b, _)| *b);
                let start = text[..start_byte].width();
                let end = start + text[start_byte..end_byte].width();
                out.push(Match { row, start, end });
                i += needle.len();
            } else {
                i += 1;
            }
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::buffer::Buffer;
    use crate::layout::Layouter;
    use crate::theme::Theme;

    fn page(text: &str) -> Page {
        let doc = crate::doc::parse(&Buffer::from_text(text));
        let style = crate::style::load("github", None).expect("style");
        Layouter::new(Theme::default_theme().expect("theme")).layout(&doc, &style, 60)
    }

    #[test]
    fn finds_every_occurrence_ignoring_case_with_cell_columns() {
        let p = page("Restic backs up. restic again, RESTIC thrice.\n");
        let m = find(&p, "restic");
        assert_eq!(m.len(), 3);
        assert_eq!((m[0].start, m[0].end), (0, 6));
        assert_eq!(m[1].start, 17);
    }

    #[test]
    fn wide_characters_count_two_columns() {
        let p = page("日本 x\n");
        let m = find(&p, "x");
        assert_eq!((m[0].start, m[0].end), (5, 6));
    }

    #[test]
    fn an_empty_query_matches_nothing() {
        assert!(find(&page("text\n"), "").is_empty());
    }
}
