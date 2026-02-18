package com.example.noteapp.ui.lists

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
