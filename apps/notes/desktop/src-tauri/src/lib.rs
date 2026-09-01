use tauri::{Manager, WebviewWindow};
use tauri_plugin_global_shortcut::{Code, Modifiers, Shortcut, ShortcutState};

/// The route the shortcut lands on. It must exist in `apps/notes/app/` — the shell owns the
/// key binding, the app owns the screen.
const CAPTURE_ROUTE: &str = "capture";

/// Ctrl+Shift+Space, and nothing else, is why this shell exists rather than a bookmark.
///
/// Commonplace is a commonplace book: the value is in catching a thought in the two seconds
/// before it is gone. A browser tab cannot be summoned from inside another application — you
/// have to find the window first, by which point you are no longer writing down the thought,
/// you are managing windows. An OS-level shortcut removes that step, and it is the single
/// capability the web export cannot provide for itself.
///
/// Space (rather than N, C, or any letter) because it is the one key no editor, terminal or
/// browser has already claimed under Ctrl+Shift on a stock Arch desktop.
fn quick_capture_shortcut() -> Shortcut {
    Shortcut::new(Some(Modifiers::CONTROL | Modifiers::SHIFT), Code::Space)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_shortcut(quick_capture_shortcut())
                .expect("Ctrl+Shift+Space is built from typed parts; it cannot fail to parse")
                .with_handler(|app, _shortcut, event| {
                    // Fires on press and release. Acting on both would open capture, then
                    // immediately re-navigate and throw away whatever the first keystroke typed.
                    if event.state() != ShortcutState::Pressed {
                        return;
                    }
                    if let Some(window) = app.get_webview_window("main") {
                        open_capture(&window);
                    }
                })
                .build(),
        )
        .run(tauri::generate_context!())
        .expect("failed to start the Commonplace window");
}

/// Surface the window from wherever it is — hidden, minimised, or behind a full-screen
/// editor — and put it on the capture screen.
fn open_capture(window: &WebviewWindow) {
    // Each of these is a no-op in the state the others handle, and any of them can fail
    // harmlessly (a compositor that refuses focus stealing, say). None of them is worth
    // killing the app over, so the shortcut degrades to "did less than you wanted".
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();

    if let Ok(mut url) = window.url() {
        // Metro serves routes path-style (`/capture`); `expo export` writes one flat file per
        // route (`capture.html`) and Tauri's asset protocol does no extension guessing, so the
        // same path is a 404 in a bundled build. `tauri dev` is a debug build and `tauri build`
        // is not, which is the cheapest honest way to tell the two apart from in here.
        let path = if cfg!(debug_assertions) {
            format!("/{CAPTURE_ROUTE}")
        } else {
            format!("/{CAPTURE_ROUTE}.html")
        };
        url.set_path(&path);

        // A full navigation, not an event the web side listens for: the export is a plain
        // Expo web build with no Tauri JS API in it, and keeping it that way is what lets the
        // same bundle run in a browser with no shell at all (ADR 0009).
        let _ = window.navigate(url);
    }
}
