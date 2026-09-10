//! Palette roles and the flavors that give them colours, read from `design/palette.toml` at build time.

use std::collections::BTreeMap;
use std::fmt;
use std::sync::LazyLock;

use serde::Deserialize;

const PALETTE: &str = include_str!("../../../../design/palette.toml");

/// A semantic colour role; styles name these, never hexes.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Role {
    Bg,
    Surface,
    Selection,
    Border,
    Fg,
    Emphasis,
    Muted,
    Accent,
    Changes,
    Float,
    Working,
    Asking,
    Blocked,
    Done,
}

impl Role {
    const ALL: [Role; 14] = [
        Role::Bg,
        Role::Surface,
        Role::Selection,
        Role::Border,
        Role::Fg,
        Role::Emphasis,
        Role::Muted,
        Role::Accent,
        Role::Changes,
        Role::Float,
        Role::Working,
        Role::Asking,
        Role::Blocked,
        Role::Done,
    ];
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Rgb(pub u8, pub u8, pub u8);

impl Rgb {
    fn parse(hex: &str) -> Option<Self> {
        let hex = hex.strip_prefix('#')?;
        if hex.len() != 6 {
            return None;
        }
        let byte = |i: usize| u8::from_str_radix(&hex[i..i + 2], 16).ok();
        Some(Self(byte(0)?, byte(2)?, byte(4)?))
    }
}

/// One flavor: every role resolved to a colour.
#[derive(Debug, Clone)]
pub struct Theme {
    pub id: String,
    pub name: String,
    pub dark: bool,
    colors: [Rgb; Role::ALL.len()],
}

impl Theme {
    #[must_use]
    pub fn color(&self, role: Role) -> Rgb {
        self.colors[role as usize]
    }

    /// The flavor with this id; the error lists what exists.
    pub fn by_id(id: &str) -> Result<&'static Theme, ThemeError> {
        let all = flavors()?;
        all.iter().find(|t| t.id == id).ok_or_else(|| ThemeError::Unknown {
            id: id.to_owned(),
            known: all.iter().map(|t| t.id.clone()).collect(),
        })
    }

    /// The palette's default flavor.
    pub fn default_theme() -> Result<&'static Theme, ThemeError> {
        let palette = PARSED.as_ref().map_err(Clone::clone)?;
        Self::by_id(&palette.default)
    }
}

#[derive(Debug, Clone, thiserror::Error)]
pub enum ThemeError {
    #[error("palette.toml is malformed: {0}")]
    Palette(String),
    #[error("theme {id:?} is not in the palette; known: {}", known.join(", "))]
    Unknown { id: String, known: Vec<String> },
}

/// Every flavor, in the palette's declared order.
pub fn flavors() -> Result<&'static [Theme], ThemeError> {
    PARSED.as_ref().map(|p| p.themes.as_slice()).map_err(Clone::clone)
}

struct Parsed {
    default: String,
    themes: Vec<Theme>,
}

static PARSED: LazyLock<Result<Parsed, ThemeError>> = LazyLock::new(|| parse(PALETTE));

#[derive(Deserialize)]
struct RawPalette {
    meta: RawMeta,
    themes: BTreeMap<String, RawTheme>,
}

#[derive(Deserialize)]
struct RawMeta {
    default: String,
    order: Vec<String>,
}

#[derive(Deserialize)]
struct RawTheme {
    name: String,
    mode: String,
    #[serde(flatten)]
    colors: BTreeMap<String, String>,
}

fn parse(text: &str) -> Result<Parsed, ThemeError> {
    let raw: RawPalette = toml::from_str(text).map_err(|e| ThemeError::Palette(e.to_string()))?;
    let mut themes = Vec::with_capacity(raw.meta.order.len());
    for id in &raw.meta.order {
        let t = raw
            .themes
            .get(id)
            .ok_or_else(|| ThemeError::Palette(format!("order names missing theme {id}")))?;
        let mut colors = [Rgb(0, 0, 0); Role::ALL.len()];
        for role in Role::ALL {
            let key = role_key(role);
            let hex = t
                .colors
                .get(key)
                .ok_or_else(|| ThemeError::Palette(format!("{id} lacks {key}")))?;
            colors[role as usize] =
                Rgb::parse(hex).ok_or_else(|| ThemeError::Palette(format!("{id}.{key} = {hex:?} is not #rrggbb")))?;
        }
        themes.push(Theme {
            id: id.clone(),
            name: t.name.clone(),
            dark: t.mode == "dark",
            colors,
        });
    }
    Ok(Parsed {
        default: raw.meta.default,
        themes,
    })
}

fn role_key(role: Role) -> &'static str {
    match role {
        Role::Bg => "bg",
        Role::Surface => "surface",
        Role::Selection => "selection",
        Role::Border => "border",
        Role::Fg => "fg",
        Role::Emphasis => "emphasis",
        Role::Muted => "muted",
        Role::Accent => "accent",
        Role::Changes => "changes",
        Role::Float => "float",
        Role::Working => "working",
        Role::Asking => "asking",
        Role::Blocked => "blocked",
        Role::Done => "done",
    }
}

impl fmt::Display for Role {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(role_key(*self))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn palette_parses_with_solarized_light_default() {
        let t = Theme::default_theme().expect("palette parses");
        assert_eq!(t.id, "solarized-light");
        assert!(!t.dark);
        assert_eq!(t.color(Role::Accent), Rgb(0x26, 0x8b, 0xd2));
        assert_eq!(flavors().expect("palette parses").len(), 4);
    }

    #[test]
    fn unknown_theme_lists_known_ids() {
        let err = Theme::by_id("nope").expect_err("unknown id fails");
        assert!(err.to_string().contains("solarized-dark"));
    }
}
