//! Code highlighting with bat's grammars and themes, one theme per flavor.

use std::sync::LazyLock;

use syntect::easy::HighlightLines;
use syntect::highlighting::{FontStyle, Theme as SyntaxTheme};
use syntect::parsing::SyntaxSet;
use syntect::util::LinesWithEndings;
use tracing::debug;
use two_face::theme::{EmbeddedLazyThemeSet, EmbeddedThemeName};

use crate::theme::{Rgb, Theme};

static SYNTAXES: LazyLock<SyntaxSet> = LazyLock::new(two_face::syntax::extra_newlines);
static THEMES: LazyLock<EmbeddedLazyThemeSet> = LazyLock::new(two_face::theme::extra);

/// One highlighted run of a code line.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Token {
    pub text: String,
    /// `None` leaves the block's own text colour.
    pub fg: Option<Rgb>,
    pub bold: bool,
    pub italic: bool,
    pub underline: bool,
}

/// Highlights code blocks with the syntax theme that matches a flavor.
#[derive(Debug, Clone, Copy)]
pub struct Highlighter {
    theme: &'static SyntaxTheme,
}

impl Highlighter {
    #[must_use]
    pub fn for_theme(theme: &Theme) -> Self {
        Self {
            theme: THEMES.get(embedded(theme)),
        }
    }

    /// Tokens per line, or `None` when no grammar knows `lang`.
    #[must_use]
    pub fn highlight(&self, lang: &str, text: &str) -> Option<Vec<Vec<Token>>> {
        let syntax = SYNTAXES.find_syntax_by_token(lang)?;
        let mut lines = HighlightLines::new(syntax, self.theme);
        let mut out = Vec::new();
        for line in LinesWithEndings::from(text) {
            let ranges = match lines.highlight_line(line, &SYNTAXES) {
                Ok(r) => r,
                Err(e) => {
                    debug!(lang, error = %e, "highlighting stopped");
                    return None;
                }
            };
            let tokens = ranges
                .into_iter()
                .map(|(style, s)| Token {
                    text: s.trim_end_matches(['\n', '\r']).to_owned(),
                    fg: (style.foreground.a != 0).then_some(Rgb(
                        style.foreground.r,
                        style.foreground.g,
                        style.foreground.b,
                    )),
                    bold: style.font_style.contains(FontStyle::BOLD),
                    italic: style.font_style.contains(FontStyle::ITALIC),
                    underline: style.font_style.contains(FontStyle::UNDERLINE),
                })
                .filter(|t| !t.text.is_empty())
                .collect();
            out.push(tokens);
        }
        Some(out)
    }
}

/// The bat theme that matches a flavor; unknown flavors fall back by light or dark mode.
fn embedded(theme: &Theme) -> EmbeddedThemeName {
    match theme.id.as_str() {
        "solarized-light" => EmbeddedThemeName::SolarizedLight,
        "solarized-dark" => EmbeddedThemeName::SolarizedDark,
        "catppuccin-latte" => EmbeddedThemeName::CatppuccinLatte,
        "catppuccin-mocha" => EmbeddedThemeName::CatppuccinMocha,
        _ if theme.dark => EmbeddedThemeName::SolarizedDark,
        _ => EmbeddedThemeName::SolarizedLight,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bash_gets_distinct_colours_per_token_kind() {
        let theme = Theme::by_id("solarized-light").expect("theme");
        let lines = Highlighter::for_theme(theme)
            .highlight("bash", "# note\nexport X=1\n")
            .expect("bash grammar");
        assert_eq!(lines.len(), 2);
        let comment = lines[0][0].fg;
        let keyword = lines[1][0].fg;
        assert!(comment.is_some() && keyword.is_some() && comment != keyword);
    }

    #[test]
    fn unknown_language_is_none() {
        let theme = Theme::by_id("solarized-light").expect("theme");
        assert!(Highlighter::for_theme(theme).highlight("no-such-lang", "x\n").is_none());
    }
}
