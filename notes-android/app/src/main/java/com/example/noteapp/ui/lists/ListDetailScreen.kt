package com.example.noteapp.ui.lists

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Image
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.TextFields
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import coil.compose.AsyncImage
import com.example.noteapp.data.NoteItem
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.LocalStrings
import com.example.noteapp.ui.theme.NeuCard
import com.example.noteapp.ui.theme.NeuTopBar

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ListDetailScreen(
    listId: String,
    viewModel: ListDetailViewModel,
    onBack: () -> Unit,
    onEditList: (String) -> Unit,
) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current
    var showAddItem by remember { mutableStateOf(false) }
    var showAddTag by remember { mutableStateOf(false) }
    var editingItem by remember { mutableStateOf<NoteItem?>(null) }

    LaunchedEffect(listId) { viewModel.loadList(listId) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(colors.bg),
    ) {
        NeuTopBar(
            title = state.list?.title ?: "List",
            onBack = onBack,
            actions = {
                IconButton(onClick = { onEditList(listId) }) {
                    Icon(Icons.Default.Edit, contentDescription = "Edit List", tint = colors.textSecondary)
                }
                IconButton(onClick = { showAddItem = true }) {
                    Icon(Icons.Default.Add, contentDescription = strings.addItem, tint = colors.accent)
                }
            },
        )

        if (state.isLoading) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = colors.accent)
            }
            return@Column
        }

        val list = state.list
        if (list == null) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(state.error ?: "List not found", color = colors.error)
            }
            return@Column
        }

        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 8.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            if (list.description.isNotBlank()) {
                item {
                    Text(
                        list.description,
                        style = TextStyle(color = colors.textSecondary, fontSize = 14.sp),
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                    )
                }
            }

            item {
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                ) {
                    list.tags.forEach { tag ->
                        NeuCard(cornerRadius = 8.dp) {
                            Row(
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(tag.name, style = TextStyle(color = colors.textPrimary, fontSize = 12.sp))
                                Spacer(Modifier.width(4.dp))
                                Icon(
                                    Icons.Default.Close,
                                    contentDescription = "Remove tag",
                                    modifier = Modifier.size(14.dp).clickable { viewModel.removeTag(listId, tag.name) },
                                    tint = colors.textSecondary,
                                )
                            }
                        }
                    }
                    Box(
                        modifier = Modifier
                            .padding(8.dp)
                            .clip(RoundedCornerShape(8.dp))
                            .background(colors.accentDim)
                            .clickable { showAddTag = true }
                            .padding(horizontal = 10.dp, vertical = 6.dp),
                    ) {
                        Text(
                            strings.addTag,
                            style = TextStyle(color = colors.accent, fontSize = 12.sp, fontWeight = FontWeight.Medium),
                        )
                    }
                }
                Spacer(Modifier.height(4.dp))
            }

            if (list.items.isEmpty()) {
                item {
                    Box(
                        Modifier.fillMaxWidth().padding(vertical = 32.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(strings.noItemsYet, color = colors.textSecondary)
                    }
                }
            }

            items(list.items, key = { it.id }) { item ->
                ItemCard(
                    item = item,
                    isFirst = list.items.first().id == item.id,
                    isLast = list.items.last().id == item.id,
                    onDelete = { viewModel.deleteItem(item.id) },
                    onEdit = { editingItem = item },
                    onMoveUp = { viewModel.moveItemUp(item.id) },
                    onMoveDown = { viewModel.moveItemDown(item.id) },
                )
            }
        }
    }

    if (showAddItem) {
        AddItemDialog(
            onDismiss = { showAddItem = false },
            onAdd = { type, content ->
                viewModel.addItem(listId, type, content)
                showAddItem = false
            },
            onUploadImage = { uri ->
                viewModel.uploadAndAddItem(listId, uri, context)
                showAddItem = false
            }
        )
    }

    if (showAddTag) {
        AddTagDialog(
            onDismiss = { showAddTag = false },
            onAdd = { tagName ->
                viewModel.addTag(listId, tagName)
                showAddTag = false
            }
        )
    }

    editingItem?.let { item ->
        EditItemDialog(
            item = item,
            onDismiss = { editingItem = null },
            onSave = { content, type ->
                viewModel.editItem(item.id, content, type)
                editingItem = null
            }
        )
    }
}

@Composable
private fun ItemCard(
    item: NoteItem,
    isFirst: Boolean,
    isLast: Boolean,
    onDelete: () -> Unit,
    onEdit: () -> Unit,
    onMoveUp: () -> Unit,
    onMoveDown: () -> Unit,
) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current
    var showMenu by remember { mutableStateOf(false) }
    var expanded by remember { mutableStateOf(false) }

    NeuCard(
        modifier = Modifier.fillMaxWidth(),
        onClick = { if (item.type != "IMAGE") expanded = !expanded },
        onLongClick = { showMenu = true },
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.Top,
        ) {
            Box(
                modifier = Modifier
                    .clip(RoundedCornerShape(8.dp))
                    .background(colors.accentDim)
                    .padding(6.dp),
            ) {
                Icon(
                    when (item.type) {
                        "IMAGE" -> Icons.Default.Image
                        "LINK" -> Icons.Default.Link
                        else -> Icons.Default.TextFields
                    },
                    contentDescription = item.type,
                    modifier = Modifier.size(16.dp),
                    tint = colors.accent,
                )
            }
            Spacer(Modifier.width(12.dp))
            if (item.type == "IMAGE") {
                AsyncImage(
                    model = item.content,
                    contentDescription = "Image",
                    modifier = Modifier
                        .weight(1f)
                        .height(160.dp)
                        .clip(RoundedCornerShape(12.dp)),
                    contentScale = ContentScale.Crop,
                )
            } else {
                Text(
                    item.content,
                    modifier = Modifier.weight(1f),
                    style = TextStyle(color = colors.textPrimary, fontSize = 14.sp),
                    maxLines = if (expanded) Int.MAX_VALUE else 3,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }

    if (showMenu) {
        ItemActionsDialog(
            isFirst = isFirst,
            isLast = isLast,
            strings = strings,
            colors = colors,
            onEdit = { onEdit(); showMenu = false },
            onDelete = { onDelete(); showMenu = false },
            onMoveUp = { onMoveUp(); showMenu = false },
            onMoveDown = { onMoveDown(); showMenu = false },
            onDismiss = { showMenu = false },
        )
    }
}

@Composable
private fun ItemActionsDialog(
    isFirst: Boolean,
    isLast: Boolean,
    strings: com.example.noteapp.ui.theme.AppStrings,
    colors: com.example.noteapp.ui.theme.AppColors,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
    onMoveUp: () -> Unit,
    onMoveDown: () -> Unit,
    onDismiss: () -> Unit,
) {
    Dialog(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(16.dp))
                .background(colors.surface),
        ) {
            if (!isFirst) {
                ActionRow(
                    icon = Icons.Default.KeyboardArrowUp,
                    label = strings.moveUp,
                    tint = colors.textPrimary,
                    onClick = onMoveUp,
                )
                RowDivider(colors)
            }
            if (!isLast) {
                ActionRow(
                    icon = Icons.Default.KeyboardArrowDown,
                    label = strings.moveDown,
                    tint = colors.textPrimary,
                    onClick = onMoveDown,
                )
                RowDivider(colors)
            }
            ActionRow(
                icon = Icons.Default.Edit,
                label = strings.editItem,
                tint = colors.textPrimary,
                onClick = onEdit,
            )
            RowDivider(colors)
            ActionRow(
                icon = Icons.Default.Delete,
                label = strings.delete,
                tint = colors.error,
                onClick = onDelete,
            )
        }
    }
}

@Composable
private fun ActionRow(icon: ImageVector, label: String, tint: Color, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 20.dp, vertical = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Icon(icon, contentDescription = label, tint = tint, modifier = Modifier.size(20.dp))
        Text(label, style = TextStyle(color = tint, fontSize = 15.sp))
    }
}

@Composable
private fun RowDivider(colors: com.example.noteapp.ui.theme.AppColors) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(1.dp)
            .background(colors.bg),
    )
}

@Composable
private fun AddItemDialog(
    onDismiss: () -> Unit,
    onAdd: (type: String, content: String) -> Unit,
    onUploadImage: (Uri) -> Unit,
) {
    val strings = LocalStrings.current
    var content by remember { mutableStateOf("") }
    var selectedType by remember { mutableStateOf("TEXT") }
    var expanded by remember { mutableStateOf(false) }
    var pickedImageUri by remember { mutableStateOf<Uri?>(null) }

    val imagePicker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) pickedImageUri = uri
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(strings.addItem) },
        text = {
            Column {
                Box {
                    TextButton(onClick = { expanded = true }) {
                        Text("Type: $selectedType")
                    }
                    DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                        listOf(strings.typeText to "TEXT", strings.typeLink to "LINK", strings.typeImage to "IMAGE").forEach { (label, type) ->
                            DropdownMenuItem(
                                text = { Text(label) },
                                onClick = {
                                    selectedType = type
                                    expanded = false
                                    if (type != "IMAGE") pickedImageUri = null
                                }
                            )
                        }
                    }
                }
                Spacer(Modifier.height(8.dp))
                if (selectedType == "IMAGE") {
                    if (pickedImageUri != null) {
                        AsyncImage(
                            model = pickedImageUri,
                            contentDescription = "Preview",
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(140.dp)
                                .clip(RoundedCornerShape(12.dp)),
                            contentScale = ContentScale.Crop,
                        )
                        Spacer(Modifier.height(8.dp))
                    }
                    TextButton(onClick = { imagePicker.launch("image/*") }) {
                        Icon(Icons.Default.Image, contentDescription = null)
                        Spacer(Modifier.width(8.dp))
                        Text(if (pickedImageUri == null) strings.pickImage else strings.changeImage)
                    }
                } else {
                    androidx.compose.material3.OutlinedTextField(
                        value = content,
                        onValueChange = { content = it },
                        label = { Text(if (selectedType == "LINK") strings.url else strings.textContent) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    if (selectedType == "IMAGE" && pickedImageUri != null) onUploadImage(pickedImageUri!!)
                    else onAdd(selectedType, content)
                },
                enabled = if (selectedType == "IMAGE") pickedImageUri != null else content.isNotBlank(),
            ) { Text(strings.add) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(strings.cancel) }
        }
    )
}

@Composable
private fun EditItemDialog(
    item: NoteItem,
    onDismiss: () -> Unit,
    onSave: (content: String, type: String) -> Unit,
) {
    val strings = LocalStrings.current
    var content by remember { mutableStateOf(item.content) }
    var selectedType by remember { mutableStateOf(item.type) }
    var expanded by remember { mutableStateOf(false) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(strings.editItem) },
        text = {
            Column {
                Box {
                    TextButton(onClick = { expanded = true }) { Text("Type: $selectedType") }
                    DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                        listOf(strings.typeText to "TEXT", strings.typeLink to "LINK", strings.typeImage to "IMAGE").forEach { (label, type) ->
                            DropdownMenuItem(
                                text = { Text(label) },
                                onClick = { selectedType = type; expanded = false }
                            )
                        }
                    }
                }
                Spacer(Modifier.height(8.dp))
                androidx.compose.material3.OutlinedTextField(
                    value = content,
                    onValueChange = { content = it },
                    label = {
                        Text(when (selectedType) { "LINK" -> strings.url; "IMAGE" -> strings.imageUrl; else -> strings.textContent })
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            TextButton(onClick = { onSave(content, selectedType) }, enabled = content.isNotBlank()) { Text(strings.save) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(strings.cancel) }
        }
    )
}

@Composable
private fun AddTagDialog(onDismiss: () -> Unit, onAdd: (String) -> Unit) {
    val strings = LocalStrings.current
    var tagName by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(strings.addTag) },
        text = {
            androidx.compose.material3.OutlinedTextField(
                value = tagName,
                onValueChange = { tagName = it },
                label = { Text(strings.addTagHint) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        },
        confirmButton = {
            TextButton(onClick = { onAdd(tagName) }, enabled = tagName.isNotBlank()) { Text(strings.add) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(strings.cancel) }
        }
    )
}
