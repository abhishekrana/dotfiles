//! Reading positions: the top source line per file, kept in the state dir so a note reopens where it was left.

use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};
use tracing::warn;

const FILE: &str = "positions.toml";
/// Entries kept; the oldest go first.
const MAX_ENTRIES: usize = 200;

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Entry {
    line: usize,
    at: u64,
}

#[derive(Debug, Default, Serialize, Deserialize)]
struct Store {
    #[serde(default)]
    entries: BTreeMap<String, Entry>,
}

#[derive(Debug)]
pub struct Positions {
    path: PathBuf,
    store: Store,
}

impl Positions {
    /// Loads the store under `dir`; a missing or unreadable file starts empty.
    #[must_use]
    pub fn open(dir: &Path) -> Self {
        let path = dir.join(FILE);
        let store = std::fs::read_to_string(&path)
            .ok()
            .and_then(|text| {
                toml::from_str(&text)
                    .map_err(|e| warn!(path = %path.display(), error = %e, "positions file ignored"))
                    .ok()
            })
            .unwrap_or_default();
        Self { path, store }
    }

    #[must_use]
    pub fn get(&self, file: &Path) -> Option<usize> {
        self.store.entries.get(&key(file)?).map(|e| e.line)
    }

    pub fn set(&mut self, file: &Path, line: usize) {
        let Some(k) = key(file) else { return };
        let at = SystemTime::now().duration_since(UNIX_EPOCH).map_or(0, |d| d.as_secs());
        self.store.entries.insert(k, Entry { line, at });
    }

    /// Writes the store, dropping the oldest entries past the cap.
    pub fn save(&mut self) -> std::io::Result<()> {
        while self.store.entries.len() > MAX_ENTRIES {
            let oldest = self
                .store
                .entries
                .iter()
                .min_by_key(|(_, e)| e.at)
                .map(|(k, _)| k.clone());
            match oldest {
                Some(k) => self.store.entries.remove(&k),
                None => break,
            };
        }
        let text = toml::to_string(&self.store).map_err(std::io::Error::other)?;
        if let Some(dir) = self.path.parent() {
            std::fs::create_dir_all(dir)?;
        }
        std::fs::write(&self.path, text)
    }
}

fn key(file: &Path) -> Option<String> {
    file.canonicalize().ok().map(|p| p.display().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn positions_round_trip_and_prune_the_oldest() {
        let dir = std::env::temp_dir().join(format!("folio-pos-{}", std::process::id()));
        std::fs::create_dir_all(&dir).expect("dir");
        let note = dir.join("note.md");
        std::fs::write(&note, "x").expect("note");
        let mut p = Positions::open(&dir);
        assert_eq!(p.get(&note), None);
        p.set(&note, 42);
        for i in 0..MAX_ENTRIES {
            let f = dir.join(format!("{i}.md"));
            std::fs::write(&f, "x").expect("file");
            p.set(&f, i);
            p.store.entries.get_mut(&key(&f).expect("key")).expect("entry").at = 1 + i as u64;
        }
        p.store.entries.get_mut(&key(&note).expect("key")).expect("entry").at = 0;
        p.save().expect("save");
        let again = Positions::open(&dir);
        assert_eq!(again.store.entries.len(), MAX_ENTRIES);
        assert_eq!(again.get(&note), None, "the oldest entry was pruned");
        assert_eq!(again.get(&dir.join("7.md")), Some(7));
        std::fs::remove_dir_all(&dir).ok();
    }
}
