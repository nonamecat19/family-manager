package com.example.noteapp.ui.lists

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.NoteList
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class ListsState(
    val myLists: List<NoteList> = emptyList(),
    val publicLists: List<NoteList> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
)

class ListsViewModel(private val listRepo: ListRepository) : ViewModel() {
    private val _state = MutableStateFlow(ListsState())
    val state = _state.asStateFlow()

    fun loadMyLists() {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            try {
                val lists = listRepo.myLists()
                _state.value = _state.value.copy(isLoading = false, myLists = lists)
            } catch (e: Exception) {
                _state.value = _state.value.copy(isLoading = false, error = e.message)
            }
        }
    }

    fun loadPublicLists() {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            try {
                val lists = listRepo.publicLists()
                _state.value = _state.value.copy(isLoading = false, publicLists = lists)
            } catch (e: Exception) {
                _state.value = _state.value.copy(isLoading = false, error = e.message)
            }
        }
    }

    fun deleteList(id: String) {
        viewModelScope.launch {
            try {
                listRepo.deleteList(id)
                _state.value = _state.value.copy(
                    myLists = _state.value.myLists.filter { it.id != id }
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }
}
