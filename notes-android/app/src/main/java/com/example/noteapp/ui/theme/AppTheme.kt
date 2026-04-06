package com.example.noteapp.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.graphics.Color

@Composable
fun AppTheme(
    colors: AppColors = AppColors(),
    strings: AppStrings = EnglishStrings,
    content: @Composable () -> Unit,
) {
    val materialColors = darkColorScheme(
        primary = colors.accent,
        onPrimary = colors.onAccent,
        background = colors.bg,
        surface = colors.surface,
        onBackground = colors.textPrimary,
        onSurface = colors.textPrimary,
        onSurfaceVariant = colors.textSecondary,
        error = colors.error,
        onError = Color.White,
        secondaryContainer = colors.accentDim,
        onSecondaryContainer = colors.accent,
        tertiaryContainer = Color(0x3314B8A6),
        onTertiaryContainer = Color(0xFF14B8A6),
    )

    CompositionLocalProvider(
        LocalAppColors provides colors,
        LocalStrings provides strings,
    ) {
        MaterialTheme(
            colorScheme = materialColors,
            typography = AppTypography,
            content = content,
        )
    }
}
