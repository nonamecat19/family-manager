package com.example.noteapp.ui.theme

import androidx.compose.runtime.Composable

// Legacy wrapper — delegates to AppTheme with defaults
@Composable
fun NoteAppTheme(content: @Composable () -> Unit) {
    AppTheme(content = content)
}
