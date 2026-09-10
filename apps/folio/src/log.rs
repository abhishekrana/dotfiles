//! Logging: a file under the state dir for the TUI, stderr for `--inline`. Level from `FOLIO_LOG`.

use std::path::PathBuf;

use tracing_appender::non_blocking::WorkerGuard;
use tracing_subscriber::EnvFilter;

const ENV: &str = "FOLIO_LOG";
/// Daily files kept before the oldest is removed.
const KEEP_LOG_FILES: usize = 3;

/// Where the log files live: `$XDG_STATE_HOME/folio` or `~/.local/state/folio`, one file a day.
#[must_use]
pub fn state_dir() -> Option<PathBuf> {
    std::env::var_os("XDG_STATE_HOME")
        .map(PathBuf::from)
        .or_else(|| std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".local").join("state")))
        .map(|d| d.join("folio"))
}

/// Installs the global subscriber. The guard flushes the file writer on drop; keep it alive in `main`.
pub fn init(to_stderr: bool) -> std::io::Result<Option<WorkerGuard>> {
    if to_stderr {
        let filter = EnvFilter::try_from_env(ENV).unwrap_or_else(|_| EnvFilter::new("warn"));
        tracing_subscriber::fmt()
            .with_env_filter(filter)
            .with_writer(std::io::stderr)
            .with_ansi(false)
            .with_target(false)
            .init();
        return Ok(None);
    }
    let Some(dir) = state_dir() else {
        return Ok(None);
    };
    std::fs::create_dir_all(&dir)?;
    let file = tracing_appender::rolling::Builder::new()
        .rotation(tracing_appender::rolling::Rotation::DAILY)
        .filename_prefix("folio")
        .filename_suffix("log")
        .max_log_files(KEEP_LOG_FILES)
        .build(&dir)
        .map_err(std::io::Error::other)?;
    let (writer, guard) = tracing_appender::non_blocking(file);
    let filter = EnvFilter::try_from_env(ENV).unwrap_or_else(|_| EnvFilter::new("info"));
    tracing_subscriber::fmt()
        .with_env_filter(filter)
        .with_writer(writer)
        .with_ansi(false)
        .init();
    Ok(Some(guard))
}
