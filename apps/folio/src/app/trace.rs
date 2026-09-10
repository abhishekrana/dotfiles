//! Action edges for the shared trace log, through the `dotfiles-trace` CLI so there is no second writer.

use std::process::{Command, Stdio};

/// Appends one `src=folio evt=<evt> k=v ...` record; silently does nothing when the CLI is absent.
pub fn edge(evt: &str, fields: &[(&str, &str)]) {
    let mut cmd = Command::new("dotfiles-trace");
    cmd.arg("log").arg("folio").arg(evt);
    for (k, v) in fields {
        cmd.arg(format!("{k}={v}"));
    }
    let _ = cmd
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn();
}
