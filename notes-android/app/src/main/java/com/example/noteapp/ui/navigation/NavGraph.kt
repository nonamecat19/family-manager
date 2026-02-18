package com.example.noteapp.ui.navigation

object Routes {
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val LISTS = "lists"
    const val LIST_DETAIL = "list/{listId}"
    const val CREATE_LIST = "create_list"

    fun listDetail(listId: String) = "list/$listId"
}
