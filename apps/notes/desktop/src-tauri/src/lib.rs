use tauri::{Manager, WebviewWindow};
use tauri_plugin_global_shortcut::{Code, Modifiers, Shortcut, ShortcutState};

const CAPTURE_ROUTE: &str = "capture";

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

fn open_capture(window: &WebviewWindow) {
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();

    if let Ok(mut url) = window.url() {
        let path = if cfg!(debug_assertions) {
            format!("/{CAPTURE_ROUTE}")
        } else {
            format!("/{CAPTURE_ROUTE}.html")
        };
        url.set_path(&path);

        let _ = window.navigate(url);
    }
}
