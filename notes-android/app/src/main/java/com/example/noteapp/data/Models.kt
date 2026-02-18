package com.example.noteapp.data

import kotlinx.serialization.Serializable

@Serializable
data class User(
    val id: String,
    val email: String,
    val username: String,
    val createdAt: String? = null,
    val updatedAt: String? = null,
)

@Serializable
data class NoteList(
    val id: String,
    val title: String,
    val description: String = "",
    val isPublic: Boolean = false,
    val owner: User? = null,
    val items: List<NoteItem> = emptyList(),
    val tags: List<Tag> = emptyList(),
    val createdAt: String? = null,
    val updatedAt: String? = null,
)

@Serializable
data class NoteItem(
    val id: String,
    val type: String, // TEXT, IMAGE, LINK
    val content: String,
    val position: Int = 0,
    val createdAt: String? = null,
    val updatedAt: String? = null,
)

@Serializable
data class Tag(
    val id: String,
    val name: String,
    val lists: List<NoteList>? = null,
    val createdAt: String? = null,
)

@Serializable
data class AuthPayload(
    val user: User,
    val accessToken: String,
    val refreshToken: String,
)
