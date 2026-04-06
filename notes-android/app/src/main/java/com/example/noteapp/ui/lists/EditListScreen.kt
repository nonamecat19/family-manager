package com.example.noteapp.ui.lists

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.LocalStrings
import com.example.noteapp.ui.theme.NeuButton
import com.example.noteapp.ui.theme.NeuTextField
import com.example.noteapp.ui.theme.NeuToggle
import com.example.noteapp.ui.theme.NeuTopBar

@Composable
fun EditListScreen(
    listId: String,
    viewModel: ListDetailViewModel,
    onBack: () -> Unit,
) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current
    val state by viewModel.state.collectAsState()

    LaunchedEffect(listId) {
        if (state.list?.id != listId) {
            viewModel.loadList(listId)
        }
    }

    val list = state.list

    var title by remember(list) { mutableStateOf(list?.title ?: "") }
    var description by remember(list) { mutableStateOf(list?.description ?: "") }
    var isPublic by remember(list) { mutableStateOf(list?.isPublic ?: false) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(colors.bg),
    ) {
        NeuTopBar(title = strings.editListTitle, onBack = onBack)

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 8.dp, vertical = 16.dp),
        ) {
            if (state.isLoading && list == null) {
                CircularProgressIndicator(color = colors.accent)
                return@Column
            }

            if (state.error != null && list == null) {
                Text(state.error!!, color = colors.error)
                return@Column
            }

            NeuTextField(
                value = title,
                onValueChange = { title = it },
                label = strings.title,
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))

            NeuTextField(
                value = description,
                onValueChange = { description = it },
                label = strings.description,
                minLines = 3,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(16.dp))

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        strings.publicList,
                        style = TextStyle(color = colors.textPrimary),
                    )
                    Text(
                        strings.anyoneCanView,
                        style = TextStyle(color = colors.textSecondary),
                    )
                }
                NeuToggle(
                    checked = isPublic,
                    onCheckedChange = { isPublic = it },
                )
            }
            Spacer(Modifier.height(24.dp))

            NeuButton(
                text = strings.saveChanges,
                onClick = {
                    viewModel.updateList(listId, title, description, isPublic) { onBack() }
                },
                enabled = title.isNotBlank() && !state.isLoading,
                loading = state.isLoading,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}
