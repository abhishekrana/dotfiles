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

const DEFAULT_INLINE_WIDTH: u16 = 80;

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
    /// Pane width for --inline.
    #[arg(long)]
    width: Option<u16>,
    /// List the styles that can be loaded and exit.
    #[arg(long)]
    list_styles: bool,
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
        return App::new(buffer, style, theme).run().context("terminal");
    }
    let document = doc::parse(&buffer);
    let width = args.width.unwrap_or(DEFAULT_INLINE_WIDTH);
    let page = Layouter::new().layout(&document, &style, width);
    let tty = io::stdout().is_terminal();
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

fn list_styles() {
    let mut names: Vec<String> = vec!["base".into(), "github".into()];
    if let Some(dir) = style::user_styles_dir()
        && let Ok(entries) = std::fs::read_dir(dir)
    {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.extension().is_some_and(|e| e == "toml")
                && let Some(stem) = path.file_stem().and_then(|s| s.to_str())
            {
                names.push(format!("{stem} (user)"));
            }
        }
    }
    for n in names {
        println!("{n}");
    }
}
