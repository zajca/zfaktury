//! Golden file testing utilities.
//!
//! Compare actual output against expected golden files.
//! Set `UPDATE_GOLDEN=1` env var to regenerate golden files.

use std::fs;
use std::path::{Path, PathBuf};

/// Assert that `actual` matches the golden file at `path`.
///
/// If `UPDATE_GOLDEN=1` is set, writes `actual` to the golden file instead.
/// On mismatch, panics with a unified diff.
pub fn assert_golden(golden_path: &Path, actual: &str) {
    if std::env::var("UPDATE_GOLDEN").unwrap_or_default() == "1" {
        if let Some(parent) = golden_path.parent() {
            fs::create_dir_all(parent).expect("failed to create golden file directory");
        }
        fs::write(golden_path, actual).expect("failed to write golden file");
        return;
    }

    let expected = match fs::read_to_string(golden_path) {
        Ok(content) => content,
        Err(e) => {
            panic!(
                "Golden file not found: {}\nRun with UPDATE_GOLDEN=1 to create it.\nError: {}",
                golden_path.display(),
                e
            );
        }
    };

    if actual != expected {
        let diff = similar::TextDiff::from_lines(expected.as_str(), actual);
        let mut diff_output = String::new();
        for change in diff.iter_all_changes() {
            let sign = match change.tag() {
                similar::ChangeTag::Delete => "-",
                similar::ChangeTag::Insert => "+",
                similar::ChangeTag::Equal => " ",
            };
            diff_output.push_str(&format!("{sign}{change}"));
        }
        panic!(
            "Golden file mismatch: {}\n\nDiff (- expected, + actual):\n{}",
            golden_path.display(),
            diff_output
        );
    }
}

/// Get the path to a golden file relative to the workspace `tests/golden/` directory.
pub fn golden_path(module: &str, name: &str) -> PathBuf {
    let workspace_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .parent()
        .unwrap()
        .to_path_buf();
    workspace_root
        .join("tests")
        .join("golden")
        .join(module)
        .join(name)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn test_assert_golden_match() {
        let dir = std::env::temp_dir().join("zfaktury-golden-test");
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("test.golden.txt");
        fs::write(&path, "hello\nworld\n").unwrap();

        assert_golden(&path, "hello\nworld\n");

        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    #[should_panic(expected = "Golden file mismatch")]
    fn test_assert_golden_mismatch() {
        let dir = std::env::temp_dir().join("zfaktury-golden-mismatch");
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("test.golden.txt");
        fs::write(&path, "expected\n").unwrap();

        assert_golden(&path, "actual\n");
    }

    #[test]
    #[should_panic(expected = "Golden file not found")]
    fn test_assert_golden_missing() {
        let path = std::env::temp_dir().join("zfaktury-nonexistent.golden.txt");
        assert_golden(&path, "anything");
    }
}
