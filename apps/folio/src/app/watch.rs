//! File watching: the parent directory is watched, so an editor that saves by rename is seen too.

use std::path::{Path, PathBuf};
use std::sync::mpsc::Sender;

use notify::{Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use tracing::debug;

use super::Input;

#[derive(Debug, thiserror::Error)]
pub enum WatchError {
    #[error("cannot watch {path}: {err}")]
    Io { path: PathBuf, err: std::io::Error },
    #[error("cannot watch {path}: {err}")]
    Notify { path: PathBuf, err: notify::Error },
}

/// Sends `Input::FileChanged` while alive; dropping it stops the watch.
#[derive(Debug)]
pub struct FileWatch {
    _watcher: RecommendedWatcher,
}

impl FileWatch {
    pub fn start(path: &Path, tx: Sender<Input>) -> Result<Self, WatchError> {
        let file = path.canonicalize().map_err(|err| WatchError::Io {
            path: path.to_owned(),
            err,
        })?;
        let dir = file.parent().map(Path::to_path_buf).ok_or_else(|| WatchError::Io {
            path: path.to_owned(),
            err: std::io::Error::other("no parent"),
        })?;
        let name = file.file_name().map(std::ffi::OsStr::to_owned);
        let mut watcher = notify::recommended_watcher(move |res: notify::Result<Event>| match res {
            Ok(ev) if concerns(&ev, name.as_deref()) => {
                debug!(kind = ?ev.kind, "file changed");
                let _ = tx.send(Input::FileChanged);
            }
            Ok(_) => {}
            Err(e) => {
                let _ = tx.send(Input::WatchError(e.to_string()));
            }
        })
        .map_err(|err| WatchError::Notify {
            path: path.to_owned(),
            err,
        })?;
        watcher
            .watch(&dir, RecursiveMode::NonRecursive)
            .map_err(|err| WatchError::Notify {
                path: path.to_owned(),
                err,
            })?;
        Ok(Self { _watcher: watcher })
    }
}

/// A write, create or rename touching the watched file name.
fn concerns(ev: &Event, name: Option<&std::ffi::OsStr>) -> bool {
    matches!(
        ev.kind,
        EventKind::Modify(_) | EventKind::Create(_) | EventKind::Remove(_)
    ) && ev.paths.iter().any(|p| p.file_name() == name)
}
