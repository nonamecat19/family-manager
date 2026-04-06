package com.example.noteapp.ui.navigation

object Routes {
    const val SPLASH = "splash"
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val LISTS = "lists"
    const val LIST_DETAIL = "list/{listId}"
    const val CREATE_LIST = "create_list"
    const val EDIT_LIST = "edit_list/{listId}"
    const val SETTINGS = "settings"

    fun listDetail(listId: String) = "list/$listId"
    fun editList(listId: String) = "edit_list/$listId"
}
