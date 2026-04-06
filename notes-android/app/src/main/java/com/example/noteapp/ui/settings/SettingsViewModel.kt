package com.example.noteapp.ui.settings

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.noteapp.data.SettingsRepository
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

class SettingsViewModel(private val repo: SettingsRepository) : ViewModel() {

    val language: StateFlow<String> = repo.language
        .stateIn(viewModelScope, SharingStarted.Eagerly, "en")

    val accentColor: StateFlow<Color> = repo.accentColor
        .map { Color(it) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, Color(0xFF8B5CF6))

    fun setLanguage(code: String) {
        viewModelScope.launch { repo.setLanguage(code) }
    }

    fun setAccentColor(color: Color) {
        viewModelScope.launch { repo.setAccentColor(color.toArgb()) }
    }
}
