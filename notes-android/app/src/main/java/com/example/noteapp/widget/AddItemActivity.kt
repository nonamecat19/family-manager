package com.example.noteapp.widget

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.lifecycle.lifecycleScope
import com.example.noteapp.data.GraphQLClient
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.TokenStore
import com.example.noteapp.ui.theme.AppColors
import com.example.noteapp.ui.theme.AppTheme
import com.example.noteapp.ui.theme.EnglishStrings
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.NeuButton
import com.example.noteapp.ui.theme.NeuCard
import com.example.noteapp.ui.theme.NeuTextField
import kotlinx.coroutines.launch

class AddItemActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val listId = intent.getStringExtra("listId") ?: run { finish(); return }

        setContent {
            AppTheme(colors = AppColors(), strings = EnglishStrings) {
                val colors = LocalAppColors.current
                var text by remember { mutableStateOf("") }
                var loading by remember { mutableStateOf(false) }

                Dialog(onDismissRequest = { finish() }) {
                    NeuCard(modifier = Modifier.fillMaxWidth().padding(16.dp)) {
                        Column(
                            modifier = Modifier.padding(16.dp),
                            verticalArrangement = Arrangement.spacedBy(12.dp),
                        ) {
                            Text("Add item", color = colors.textPrimary)
                            NeuTextField(
                                value = text,
                                onValueChange = { text = it },
                                label = "Item text",
                                singleLine = true,
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                NeuButton(
                                    text = "Cancel",
                                    onClick = { finish() },
                                    modifier = Modifier.weight(1f),
                                )
                                NeuButton(
                                    text = "Add",
                                    enabled = text.isNotBlank(),
                                    loading = loading,
                                    onClick = {
                                        loading = true
                                        lifecycleScope.launch {
                                            try {
                                                val tokenStore = TokenStore(applicationContext)
                                                val client = GraphQLClient(WIDGET_BASE_URL, tokenStore)
                                                val repo = ListRepository(client)
                                                repo.createItem(listId, "TEXT", text.trim())
                                                val updatedList = repo.getList(listId)
                                                if (updatedList != null) {
                                                    val items = updatedList.items
                                                        .filter { it.type == "TEXT" }
                                                        .map { WidgetItem(it.id, it.content) }
                                                    ListWidgetReceiver.refreshAllForList(
                                                        applicationContext, listId, updatedList.title, items,
                                                    )
                                                }
                                            } catch (_: Exception) {}
                                            finish()
                                        }
                                    },
                                    modifier = Modifier.weight(1f),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
