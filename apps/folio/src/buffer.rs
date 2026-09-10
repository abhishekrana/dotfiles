//! The text: one rope, the only copy of the document.

use std::fs::File;
use std::io::{self, BufReader, Read};
use std::path::{Path, PathBuf};

use ropey::Rope;

#[derive(Debug, thiserror::Error)]
pub enum BufferError {
    #[error("cannot read {path}: {err}")]
    Read { path: PathBuf, err: io::Error },
    #[error("cannot read stdin: {0}")]
    Stdin(io::Error),
}

/// Document text plus where it came from.
#[derive(Debug, Clone)]
pub struct Buffer {
    rope: Rope,
    path: Option<PathBuf>,
}

impl Buffer {
    pub fn from_path(path: &Path) -> Result<Self, BufferError> {
        let file = File::open(path).map_err(|err| BufferError::Read {
            path: path.to_owned(),
            err,
        })?;
        let rope = Rope::from_reader(BufReader::new(file)).map_err(|err| BufferError::Read {
            path: path.to_owned(),
            err,
        })?;
        Ok(Self {
            rope,
            path: Some(path.to_owned()),
        })
    }

    pub fn from_reader(reader: impl Read) -> Result<Self, BufferError> {
        let rope = Rope::from_reader(BufReader::new(reader)).map_err(BufferError::Stdin)?;
        Ok(Self { rope, path: None })
    }

    #[must_use]
    pub fn from_text(text: &str) -> Self {
        Self {
            rope: Rope::from_str(text),
            path: None,
        }
    }

    /// Re-reads the file this buffer was loaded from; a buffer without a path is unchanged.
    pub fn reload(&mut self) -> Result<(), BufferError> {
        if let Some(path) = self.path.clone() {
            *self = Self::from_path(&path)?;
        }
        Ok(())
    }

    #[must_use]
    pub fn path(&self) -> Option<&Path> {
        self.path.as_deref()
    }

    #[must_use]
    pub fn len_bytes(&self) -> usize {
        self.rope.len_bytes()
    }

    /// The whole text, contiguous, for the parser.
    #[must_use]
    pub fn text(&self) -> String {
        self.rope.to_string()
    }

    /// Byte offset of a zero-based line; `None` past the end.
    #[must_use]
    pub fn line_to_byte(&self, line: usize) -> Option<usize> {
        (line < self.rope.len_lines()).then(|| self.rope.line_to_byte(line))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn line_offsets_follow_newlines() {
        let b = Buffer::from_text("ab\ncd\n");
        assert_eq!(b.line_to_byte(0), Some(0));
        assert_eq!(b.line_to_byte(1), Some(3));
        assert_eq!(b.line_to_byte(2), Some(6));
        assert_eq!(b.line_to_byte(3), None);
    }
}
