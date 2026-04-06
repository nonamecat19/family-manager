package com.example.noteapp.ui.lists

import android.content.Context
import android.net.Uri
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.NoteList
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class ListDetailState(
    val list: NoteList? = null,
    val isLoading: Boolean = false,
    val error: String? = null,
)

class ListDetailViewModel(private val listRepo: ListRepository) : ViewModel() {
    private val _state = MutableStateFlow(ListDetailState())
    val state = _state.asStateFlow()

    fun loadList(id: String) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            try {
                val list = listRepo.getList(id)
                _state.value = _state.value.copy(isLoading = false, list = list)
            } catch (e: Exception) {
                _state.value = _state.value.copy(isLoading = false, error = e.message)
            }
        }
    }

    fun addItem(listId: String, type: String, content: String) {
        viewModelScope.launch {
            try {
                val position = _state.value.list?.items?.size ?: 0
                listRepo.createItem(listId, type, content, position)
                loadList(listId)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun uploadAndAddItem(listId: String, uri: Uri, context: Context) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true)
            try {
                val imageUrl = listRepo.uploadImage(context, uri)
                val position = _state.value.list?.items?.size ?: 0
                listRepo.createItem(listId, "IMAGE", imageUrl, position)
                loadList(listId)
            } catch (e: Exception) {
                _state.value = _state.value.copy(isLoading = false, error = e.message)
            }
        }
    }

    fun editItem(itemId: String, content: String, type: String) {
        viewModelScope.launch {
            try {
                listRepo.updateItem(itemId, content, type)
                val listId = _state.value.list?.id ?: return@launch
                loadList(listId)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun deleteItem(itemId: String) {
        viewModelScope.launch {
            try {
                listRepo.deleteItem(itemId)
                val listId = _state.value.list?.id ?: return@launch
                loadList(listId)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun moveItemUp(itemId: String) {
        val list = _state.value.list ?: return
        val items = list.items.toMutableList()
        val idx = items.indexOfFirst { it.id == itemId }
        if (idx <= 0) return
        val swapped = items.toMutableList()
        val tmp = swapped[idx]
        swapped[idx] = swapped[idx - 1]
        swapped[idx - 1] = tmp
        val newOrder = swapped.map { it.id }
        viewModelScope.launch {
            try {
                val reordered = listRepo.reorderItems(list.id, newOrder)
                _state.value = _state.value.copy(list = list.copy(items = reordered))
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun moveItemDown(itemId: String) {
        val list = _state.value.list ?: return
        val items = list.items.toMutableList()
        val idx = items.indexOfFirst { it.id == itemId }
        if (idx < 0 || idx >= items.size - 1) return
        val swapped = items.toMutableList()
        val tmp = swapped[idx]
        swapped[idx] = swapped[idx + 1]
        swapped[idx + 1] = tmp
        val newOrder = swapped.map { it.id }
        viewModelScope.launch {
            try {
                val reordered = listRepo.reorderItems(list.id, newOrder)
                _state.value = _state.value.copy(list = list.copy(items = reordered))
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun updateList(id: String, title: String, description: String, isPublic: Boolean, onDone: () -> Unit) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            try {
                val updated = listRepo.updateList(id, title, description, isPublic)
                _state.value = _state.value.copy(isLoading = false, list = updated)
                onDone()
            } catch (e: Exception) {
                _state.value = _state.value.copy(isLoading = false, error = e.message)
            }
        }
    }

    fun addTag(listId: String, tagName: String) {
        viewModelScope.launch {
            try {
                val list = listRepo.addTag(listId, tagName)
                _state.value = _state.value.copy(list = list)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun removeTag(listId: String, tagName: String) {
        viewModelScope.launch {
            try {
                val list = listRepo.removeTag(listId, tagName)
                _state.value = _state.value.copy(list = list)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }
}
