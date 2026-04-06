package com.example.noteapp.ui.lists

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
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
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.SwipeToDismissBox
import androidx.compose.material3.SwipeToDismissBoxValue
import androidx.compose.material3.Text
import androidx.compose.material3.rememberSwipeToDismissBoxState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.noteapp.data.NoteList
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.LocalStrings
import com.example.noteapp.ui.theme.NeuButton
import com.example.noteapp.ui.theme.NeuCard
import com.example.noteapp.ui.theme.NeuSegmentedControl

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun ListsScreen(
    viewModel: ListsViewModel,
    onListClick: (String) -> Unit,
    onCreateList: () -> Unit,
    onSettings: () -> Unit,
) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current
    val state by viewModel.state.collectAsState()
    var selectedTab by remember { mutableIntStateOf(0) }

    LaunchedEffect(selectedTab) {
        if (selectedTab == 0) viewModel.loadMyLists() else viewModel.loadPublicLists()
    }

    val lists = if (selectedTab == 0) state.myLists else state.publicLists

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(colors.bg),
    ) {
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 100.dp),
        ) {
            // Minimal top row — settings gear only
            item {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(colors.bg)
                        .padding(top = 40.dp, end = 8.dp),
                    horizontalArrangement = Arrangement.End,
                ) {
                    IconButton(onClick = onSettings) {
                        Icon(
                            Icons.Default.Settings,
                            contentDescription = strings.settings,
                            tint = colors.textSecondary,
                        )
                    }
                }
            }

            // Tab control
            item {
                NeuSegmentedControl(
                    options = listOf(strings.myLists, strings.public),
                    selected = selectedTab,
                    onSelect = { selectedTab = it },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(8.dp))
            }

            // Content
            if (state.isLoading) {
                item {
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .padding(vertical = 48.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        CircularProgressIndicator(color = colors.accent)
                    }
                }
            } else if (state.error != null) {
                item {
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .padding(vertical = 48.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(state.error!!, color = colors.error)
                    }
                }
            } else if (lists.isEmpty()) {
                item {
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .padding(vertical = 48.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            if (selectedTab == 0) strings.noListsYet else strings.noPublicLists,
                            color = colors.textSecondary,
                        )
                    }
                }
            } else {
                items(lists, key = { it.id }) { list ->
                    if (selectedTab == 0) {
                        SwipeToDeleteListCard(
                            list = list,
                            onClick = { onListClick(list.id) },
                            onDelete = { viewModel.deleteList(list.id) },
                        )
                    } else {
                        ListCard(list = list, onClick = { onListClick(list.id) })
                    }
                }
            }
        }

        // FAB
        if (selectedTab == 0) {
            Box(
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(24.dp),
            ) {
                NeuButton(
                    text = "+ ${strings.newList}",
                    onClick = onCreateList,
                    modifier = Modifier.width(160.dp),
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SwipeToDeleteListCard(list: NoteList, onClick: () -> Unit, onDelete: () -> Unit) {
    val colors = LocalAppColors.current
    val dismissState = rememberSwipeToDismissBoxState(
        confirmValueChange = { value ->
            if (value == SwipeToDismissBoxValue.EndToStart) {
                onDelete()
                true
            } else false
        }
    )

    SwipeToDismissBox(
        state = dismissState,
        enableDismissFromStartToEnd = false,
        backgroundContent = {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 16.dp, vertical = 4.dp)
                    .clip(RoundedCornerShape(16.dp))
                    .background(colors.error)
                    .padding(end = 20.dp),
                contentAlignment = Alignment.CenterEnd,
            ) {
                Icon(
                    Icons.Default.Delete,
                    contentDescription = "Delete",
                    tint = colors.onAccent,
                )
            }
        },
    ) {
        ListCard(list = list, onClick = onClick)
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ListCard(list: NoteList, onClick: () -> Unit) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current

    NeuCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 8.dp, vertical = 0.dp),
        onClick = onClick,
    ) {
        Row(modifier = Modifier.fillMaxWidth().height(IntrinsicSize.Max)) {
            // Left accent bar
            Box(
                modifier = Modifier
                    .width(4.dp)
                    .fillMaxHeight()
                    .background(
                        if (list.isPublic) colors.accent.copy(alpha = 0.6f)
                        else colors.accent
                    )
            )
            Column(
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 12.dp, vertical = 12.dp)
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        list.title,
                        style = TextStyle(
                            color = colors.textPrimary,
                            fontWeight = FontWeight.SemiBold,
                            fontSize = 16.sp,
                        ),
                        modifier = Modifier.weight(1f),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    if (list.isPublic) {
                        Spacer(Modifier.width(8.dp))
                        Box(
                            modifier = Modifier
                                .clip(RoundedCornerShape(6.dp))
                                .background(colors.accentDim)
                                .padding(horizontal = 8.dp, vertical = 2.dp),
                        ) {
                            Text(
                                strings.publicBadge,
                                style = TextStyle(
                                    color = colors.accent,
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.Medium,
                                ),
                            )
                        }
                    }
                }

                if (list.description.isNotBlank()) {
                    Spacer(Modifier.height(4.dp))
                    Text(
                        list.description,
                        style = TextStyle(color = colors.textSecondary, fontSize = 13.sp),
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }

                if (list.items.isNotEmpty() || list.tags.isNotEmpty()) {
                    Spacer(Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        if (list.items.isNotEmpty()) {
                            Box(
                                modifier = Modifier
                                    .clip(RoundedCornerShape(6.dp))
                                    .background(colors.accentDim)
                                    .padding(horizontal = 8.dp, vertical = 2.dp),
                            ) {
                                Text(
                                    "${list.items.size} item${if (list.items.size != 1) "s" else ""}",
                                    style = TextStyle(color = colors.accent, fontSize = 11.sp),
                                )
                            }
                        }
                        list.tags.take(3).forEach { tag ->
                            Box(
                                modifier = Modifier
                                    .clip(RoundedCornerShape(6.dp))
                                    .background(colors.surface)
                                    .padding(horizontal = 8.dp, vertical = 2.dp),
                            ) {
                                Text(
                                    tag.name,
                                    style = TextStyle(color = colors.textSecondary, fontSize = 11.sp),
                                )
                            }
                        }
                        if (list.tags.size > 3) {
                            Text(
                                "+${list.tags.size - 3}",
                                style = TextStyle(color = colors.textSecondary, fontSize = 11.sp),
                            )
                        }
                    }
                }

                if (list.owner != null) {
                    Spacer(Modifier.height(4.dp))
                    Text(
                        "by ${list.owner.username}",
                        style = TextStyle(color = colors.textSecondary, fontSize = 11.sp),
                    )
                }
            }
        }
    }
}
