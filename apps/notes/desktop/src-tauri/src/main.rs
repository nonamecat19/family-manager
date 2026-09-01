// Prevents a second console window from opening alongside the app on Windows release builds.
// Arch is the only target we bundle for, but the attribute costs nothing and removing it is
// the kind of thing nobody remembers to do.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    commonplace_lib::run()
}
