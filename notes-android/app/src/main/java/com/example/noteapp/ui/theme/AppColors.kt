package com.example.noteapp.ui.theme

import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

data class AppColors(
    val bg: Color = Color(0xFF1C1C2E),
    val surface: Color = Color(0xFF222238),
    val textPrimary: Color = Color.White,
    val textSecondary: Color = Color(0xFF888899),
    val accent: Color = Color(0xFF8B5CF6),
    val accentDim: Color = Color(0x1A8B5CF6),
    val error: Color = Color(0xFFFF4949),
    val onAccent: Color = Color.White,
)

val LocalAppColors = staticCompositionLocalOf { AppColors() }
