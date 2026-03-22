use std::borrow::Cow;

use gpui::{AssetSource, Result, SharedString};

/// Embedded asset source that includes all icon SVGs at compile time.
pub struct EmbeddedAssets;

impl AssetSource for EmbeddedAssets {
    fn load(&self, path: &str) -> Result<Option<Cow<'static, [u8]>>> {
        let data: Option<&'static [u8]> = match path {
            "icons/home.svg" => Some(include_bytes!("../assets/icons/home.svg")),
            "icons/bar-chart.svg" => Some(include_bytes!("../assets/icons/bar-chart.svg")),
            "icons/document-text.svg" => Some(include_bytes!("../assets/icons/document-text.svg")),
            "icons/refresh-cw.svg" => Some(include_bytes!("../assets/icons/refresh-cw.svg")),
            "icons/credit-card.svg" => Some(include_bytes!("../assets/icons/credit-card.svg")),
            "icons/users.svg" => Some(include_bytes!("../assets/icons/users.svg")),
            "icons/grid.svg" => Some(include_bytes!("../assets/icons/grid.svg")),
            "icons/bookmark.svg" => Some(include_bytes!("../assets/icons/bookmark.svg")),
            "icons/shield-check.svg" => Some(include_bytes!("../assets/icons/shield-check.svg")),
            "icons/calendar.svg" => Some(include_bytes!("../assets/icons/calendar.svg")),
            "icons/trending-up.svg" => Some(include_bytes!("../assets/icons/trending-up.svg")),
            "icons/building.svg" => Some(include_bytes!("../assets/icons/building.svg")),
            "icons/envelope.svg" => Some(include_bytes!("../assets/icons/envelope.svg")),
            "icons/hash.svg" => Some(include_bytes!("../assets/icons/hash.svg")),
            "icons/tag.svg" => Some(include_bytes!("../assets/icons/tag.svg")),
            "icons/document.svg" => Some(include_bytes!("../assets/icons/document.svg")),
            "icons/upload.svg" => Some(include_bytes!("../assets/icons/upload.svg")),
            "icons/clipboard-check.svg" => {
                Some(include_bytes!("../assets/icons/clipboard-check.svg"))
            }
            "icons/database.svg" => Some(include_bytes!("../assets/icons/database.svg")),
            "icons/plus.svg" => Some(include_bytes!("../assets/icons/plus.svg")),
            "icons/arrow-left.svg" => Some(include_bytes!("../assets/icons/arrow-left.svg")),
            "icons/arrow-right.svg" => Some(include_bytes!("../assets/icons/arrow-right.svg")),
            _ => None,
        };
        Ok(data.map(Cow::Borrowed))
    }

    fn list(&self, _path: &str) -> Result<Vec<SharedString>> {
        Ok(vec![])
    }
}
