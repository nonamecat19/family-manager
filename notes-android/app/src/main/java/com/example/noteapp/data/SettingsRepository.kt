package com.example.noteapp.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.intPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

private val Context.settingsDataStore by preferencesDataStore(name = "settings")

class SettingsRepository(private val context: Context) {
    private val languageKey = stringPreferencesKey("language")
    private val accentColorKey = intPreferencesKey("accent_color")

    val language: Flow<String> = context.settingsDataStore.data.map {
        it[languageKey] ?: "en"
    }

    val accentColor: Flow<Int> = context.settingsDataStore.data.map {
        it[accentColorKey] ?: 0xFF8B5CF6.toInt()
    }

    suspend fun setLanguage(code: String) {
        context.settingsDataStore.edit { it[languageKey] = code }
    }

    suspend fun setAccentColor(argb: Int) {
        context.settingsDataStore.edit { it[accentColorKey] = argb }
    }
}
