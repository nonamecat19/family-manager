package com.example.noteapp.ui.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.LocalStrings
import com.example.noteapp.ui.theme.NeuSegmentedControl
import com.example.noteapp.ui.theme.NeuTopBar

private val accentOptions = listOf(
    Color(0xFF8B5CF6), // Purple
    Color(0xFF3B82F6), // Blue
    Color(0xFF14B8A6), // Teal
    Color(0xFFF43F5E), // Rose
    Color(0xFFF97316), // Orange
    Color(0xFFEAB308), // Amber
)

@Composable
fun SettingsScreen(
    viewModel: SettingsViewModel,
    onBack: () -> Unit,
    onLogout: () -> Unit,
) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current
    val language by viewModel.language.collectAsState()
    val accentColor by viewModel.accentColor.collectAsState()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(colors.bg),
    ) {
        NeuTopBar(title = strings.settings, onBack = onBack)

        Spacer(Modifier.height(8.dp))

        // Language
        Text(
            text = strings.language,
            color = colors.textSecondary,
            style = TextStyle(fontSize = 12.sp, fontWeight = FontWeight.Medium),
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
        )
        NeuSegmentedControl(
            options = listOf(strings.languageEn, strings.languageUk),
            selected = if (language == "en") 0 else 1,
            onSelect = { viewModel.setLanguage(if (it == 0) "en" else "uk") },
            modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp),
        )

        Spacer(Modifier.height(24.dp))

        // Accent color
        Text(
            text = strings.accentColor,
            color = colors.textSecondary,
            style = TextStyle(fontSize = 12.sp, fontWeight = FontWeight.Medium),
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            accentOptions.forEach { color ->
                val isSelected = accentColor == color
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(color)
                        .then(if (isSelected) Modifier.border(2.dp, Color.White, CircleShape) else Modifier)
                        .clickable { viewModel.setAccentColor(color) },
                    contentAlignment = Alignment.Center,
                ) {
                    if (isSelected) {
                        Icon(
                            Icons.Default.Check,
                            contentDescription = null,
                            tint = Color.White,
                            modifier = Modifier.size(18.dp),
                        )
                    }
                }
            }
        }

        Spacer(Modifier.weight(1f))

        // Logout
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(onClick = onLogout)
                .padding(horizontal = 20.dp, vertical = 18.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Icon(
                Icons.AutoMirrored.Filled.Logout,
                contentDescription = strings.logout,
                tint = colors.error,
                modifier = Modifier.size(20.dp),
            )
            Text(
                strings.logout,
                style = TextStyle(color = colors.error, fontSize = 15.sp),
            )
        }
    }
}
