//! CLI: open a file or stdin in the reader, or render it inline to stdout.

use std::io::{self, IsTerminal, Write};
use std::path::PathBuf;
use std::process::ExitCode;

use anyhow::{Context, bail};
use clap::{Parser, ValueEnum};
use tracing::{error, info};

use folio::app::App;
use folio::buffer::Buffer;
use folio::layout::Layouter;
use folio::style::{self, StyleError};
use folio::theme::Theme;
use folio::{doc, render};

/// Width for --inline when neither a flag, fzf nor a terminal says otherwise.
const DEFAULT_INLINE_WIDTH: u16 = 120;
/// fzf exports its preview pane's width here.
const FZF_WIDTH_ENV: &str = "FZF_PREVIEW_COLUMNS";

/// A markdown reader for the terminal that reads like a page.
#[derive(Debug, Parser)]
#[command(name = "folio", version, about)]
struct Args {
    /// Markdown file; stdin when absent and not a terminal.
    file: Option<PathBuf>,
    /// Render to stdout instead of opening the reader.
    #[arg(long)]
    inline: bool,
    /// Output for --inline: auto is ansi on a terminal and plain in a pipe.
    #[arg(long, value_enum, default_value_t = Format::Auto, requires = "inline")]
    format: Format,
    /// Style name: a built-in or a file in ~/.config/folio/styles.
    #[arg(long, default_value = "github")]
    style: String,
    /// Theme flavor from the palette.
    #[arg(long, env = "FOLIO_THEME")]
    theme: Option<String>,
    /// Pane width for --inline; defaults to fzf's preview width, else the terminal's, else 120.
    #[arg(long)]
    width: Option<u16>,
    /// List the styles that can be loaded and exit.
    #[arg(long)]
    list_styles: bool,
    /// Do not follow the file on disk (the reader reloads on change by default).
    #[arg(long)]
    no_watch: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, ValueEnum)]
enum Format {
    Auto,
    Ansi,
    Plain,
}

fn main() -> ExitCode {
    let args = Args::parse();
    let to_stderr = args.inline || args.list_styles;
    let _guard = match folio::log::init(to_stderr) {
        Ok(g) => g,
        Err(e) => {
            eprintln!("folio: cannot open the log: {e}");
            None
        }
    };
    match run(&args) {
        Ok(()) => ExitCode::SUCCESS,
        Err(e) => {
            if !to_stderr {
                error!(error = format!("{e:#}"), "exit");
            }
            eprintln!("folio: {e:#}");
            let style_error = e.downcast_ref::<StyleError>().is_some();
            ExitCode::from(if style_error { 2 } else { 1 })
        }
    }
}

fn run(args: &Args) -> anyhow::Result<()> {
    if args.list_styles {
        list_styles();
        return Ok(());
    }
    let theme = match &args.theme {
        Some(id) => Theme::by_id(id)?,
        None => Theme::default_theme()?,
    };
    let style = style::load(&args.style, style::user_styles_dir().as_deref())?;
    let buffer = match &args.file {
        Some(path) => Buffer::from_path(path)?,
        None if io::stdin().is_terminal() => bail!("no file given and stdin is a terminal (see --help)"),
        None => Buffer::from_reader(io::stdin().lock())?,
    };
    info!(path = ?buffer.path(), bytes = buffer.len_bytes(), style = %style.name, theme = %theme.id, "loaded");

    if !args.inline {
        return App::new(buffer, style, theme)
            .with_watch(!args.no_watch)
            .run()
            .context("terminal");
    }
    let document = doc::parse(&buffer);
    let tty = io::stdout().is_terminal();
    let terminal_cols = tty.then(|| crossterm::terminal::size().ok().map(|(w, _)| w)).flatten();
    let width = inline_width(args.width, std::env::var(FZF_WIDTH_ENV).ok().as_deref(), terminal_cols);
    let page = Layouter::new(theme).layout(&document, &style, width);
    let ansi = match args.format {
        Format::Ansi => true,
        Format::Plain => false,
        Format::Auto => tty,
    };
    let text = if ansi {
        render::ansi(&page, theme, tty)
    } else {
        render::plain(&page)
    };
    io::stdout()
        .lock()
        .write_all(text.as_bytes())
        .context("writing to stdout")
}

/// The flag wins, then fzf's preview width, then the terminal, then the default.
fn inline_width(flag: Option<u16>, fzf: Option<&str>, terminal: Option<u16>) -> u16 {
    flag.or_else(|| fzf.and_then(|v| v.trim().parse().ok()))
        .or(terminal)
        .filter(|w| *w > 0)
        .unwrap_or(DEFAULT_INLINE_WIDTH)
}

/// Built-ins, then user styles; a user file that shadows a built-in says so.
fn list_styles() {
    let mut user: Vec<String> = Vec::new();
    if let Some(dir) = style::user_styles_dir()
        && let Ok(entries) = std::fs::read_dir(dir)
    {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.extension().is_some_and(|e| e == "toml")
                && let Some(stem) = path.file_stem().and_then(|s| s.to_str())
            {
                user.push(stem.to_owned());
            }
        }
    }
    user.sort();
    for name in style::builtins() {
        let shadowed = user.iter().any(|u| u == name);
        println!("{name}{}", if shadowed { "  (replaced by the user file)" } else { "" });
    }
    for name in user.iter().filter(|u| !style::builtins().any(|b| b == u.as_str())) {
        println!("{name}  (user)");
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn inline_width_prefers_flag_then_fzf_then_terminal() {
        assert_eq!(inline_width(Some(72), Some("90"), Some(200)), 72);
        assert_eq!(inline_width(None, Some("90"), Some(200)), 90);
        assert_eq!(inline_width(None, Some("nope"), Some(200)), 200);
        assert_eq!(inline_width(None, None, None), DEFAULT_INLINE_WIDTH);
        assert_eq!(inline_width(None, Some("0"), None), DEFAULT_INLINE_WIDTH);
    }
}
