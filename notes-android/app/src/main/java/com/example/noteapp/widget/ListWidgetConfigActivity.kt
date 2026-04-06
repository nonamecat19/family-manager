package com.example.noteapp.widget

import android.appwidget.AppWidgetManager
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.lifecycleScope
import com.example.noteapp.data.GraphQLClient
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.NoteList
import com.example.noteapp.data.TokenStore
import com.example.noteapp.ui.theme.AppColors
import com.example.noteapp.ui.theme.AppTheme
import com.example.noteapp.ui.theme.EnglishStrings
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.NeuCard
import kotlinx.coroutines.launch

class ListWidgetConfigActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val appWidgetId = intent.extras?.getInt(
            AppWidgetManager.EXTRA_APPWIDGET_ID,
            AppWidgetManager.INVALID_APPWIDGET_ID,
        ) ?: AppWidgetManager.INVALID_APPWIDGET_ID

        setResult(RESULT_CANCELED)

        setContent {
            AppTheme(colors = AppColors(), strings = EnglishStrings) {
                val colors = LocalAppColors.current
                var lists by remember { mutableStateOf<List<NoteList>>(emptyList()) }
                var loading by remember { mutableStateOf(true) }
                var error by remember { mutableStateOf(false) }

                LaunchedEffect(Unit) {
                    try {
                        val tokenStore = TokenStore(applicationContext)
                        val client = GraphQLClient(WIDGET_BASE_URL, tokenStore)
                        lists = ListRepository(client).myLists()
                    } catch (_: Exception) {
                        error = true
                    }
                    loading = false
                }

                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(colors.bg)
                        .padding(top = 48.dp),
                ) {
                    Text(
                        text = "Choose a list for the widget",
                        color = colors.textPrimary,
                        modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp),
                    )
                    when {
                        loading -> CircularProgressIndicator(
                            modifier = Modifier.padding(24.dp),
                            color = colors.accent,
                        )
                        error -> Text(
                            text = "Not logged in. Open the app first.",
                            color = colors.error,
                            modifier = Modifier.padding(horizontal = 20.dp),
                        )
                        else -> LazyColumn {
                            items(lists) { list ->
                                NeuCard(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(horizontal = 12.dp, vertical = 4.dp),
                                    onClick = {
                                        lifecycleScope.launch {
                                            try {
                                                val tokenStore = TokenStore(applicationContext)
                                                val client = GraphQLClient(WIDGET_BASE_URL, tokenStore)
                                                val fullList = ListRepository(client).getList(list.id)
                                                val items = fullList?.items
                                                    ?.filter { it.type == "TEXT" }
                                                    ?.map { WidgetItem(it.id, it.content) }
                                                    ?: emptyList()
                                                WidgetPreferences.setData(
                                                    applicationContext, appWidgetId,
                                                    list.id, list.title, items,
                                                )
                                            } catch (_: Exception) {
                                                WidgetPreferences.setData(
                                                    applicationContext, appWidgetId,
                                                    list.id, list.title, emptyList(),
                                                )
                                            }
                                            val manager = AppWidgetManager.getInstance(applicationContext)
                                            ListWidgetReceiver.updateWidget(applicationContext, manager, appWidgetId)
                                            setResult(
                                                RESULT_OK,
                                                Intent().putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, appWidgetId),
                                            )
                                            finish()
                                        }
                                    },
                                ) {
                                    Text(
                                        text = list.title,
                                        color = colors.textPrimary,
                                        modifier = Modifier.padding(16.dp),
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
