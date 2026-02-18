package com.example.noteapp.data

import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

class ListRepository(private val client: GraphQLClient) {

    private val listFields = """
        id title description isPublic
        owner { id email username }
        items { id type content position createdAt }
        tags { id name }
        createdAt updatedAt
    """.trimIndent()

    suspend fun myLists(): List<NoteList> {
        val response = client.execute(
            query = "query { myLists { $listFields } }"
        )
        return client.decode(client.extractData(response, "myLists"))
    }

    suspend fun publicLists(limit: Int = 20, offset: Int = 0): List<NoteList> {
        val response = client.execute(
            query = """
                query PublicLists(${'$'}limit: Int, ${'$'}offset: Int) {
                    publicLists(limit: ${'$'}limit, offset: ${'$'}offset) { $listFields }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("limit", limit)
                put("offset", offset)
            }
        )
        return client.decode(client.extractData(response, "publicLists"))
    }

    suspend fun getList(id: String): NoteList {
        val response = client.execute(
            query = """
                query GetList(${'$'}id: ID!) {
                    list(id: ${'$'}id) { $listFields }
                }
            """.trimIndent(),
            variables = buildJsonObject { put("id", id) }
        )
        return client.decode(client.extractData(response, "list"))
    }

    suspend fun createList(title: String, description: String, isPublic: Boolean): NoteList {
        val response = client.execute(
            query = """
                mutation CreateList(${'$'}title: String!, ${'$'}desc: String, ${'$'}public: Boolean) {
                    createList(input: { title: ${'$'}title, description: ${'$'}desc, isPublic: ${'$'}public }) { $listFields }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("title", title)
                put("desc", description)
                put("public", isPublic)
            }
        )
        return client.decode(client.extractData(response, "createList"))
    }

    suspend fun updateList(id: String, title: String?, description: String?, isPublic: Boolean?): NoteList {
        val response = client.execute(
            query = """
                mutation UpdateList(${'$'}id: ID!, ${'$'}title: String, ${'$'}desc: String, ${'$'}public: Boolean) {
                    updateList(id: ${'$'}id, input: { title: ${'$'}title, description: ${'$'}desc, isPublic: ${'$'}public }) { $listFields }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("id", id)
                if (title != null) put("title", title)
                if (description != null) put("desc", description)
                if (isPublic != null) put("public", isPublic)
            }
        )
        return client.decode(client.extractData(response, "updateList"))
    }

    suspend fun deleteList(id: String): Boolean {
        val response = client.execute(
            query = """
                mutation DeleteList(${'$'}id: ID!) { deleteList(id: ${'$'}id) }
            """.trimIndent(),
            variables = buildJsonObject { put("id", id) }
        )
        return client.decode(client.extractData(response, "deleteList"))
    }

    suspend fun createItem(listId: String, type: String, content: String, position: Int? = null): NoteItem {
        val response = client.execute(
            query = """
                mutation CreateItem(${'$'}listId: ID!, ${'$'}type: ItemType!, ${'$'}content: String!, ${'$'}position: Int) {
                    createItem(input: { listID: ${'$'}listId, type: ${'$'}type, content: ${'$'}content, position: ${'$'}position }) {
                        id type content position createdAt
                    }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("listId", listId)
                put("type", type)
                put("content", content)
                if (position != null) put("position", position)
            }
        )
        return client.decode(client.extractData(response, "createItem"))
    }

    suspend fun updateItem(id: String, content: String?, type: String?): NoteItem {
        val response = client.execute(
            query = """
                mutation UpdateItem(${'$'}id: ID!, ${'$'}content: String, ${'$'}type: ItemType) {
                    updateItem(id: ${'$'}id, input: { content: ${'$'}content, type: ${'$'}type }) {
                        id type content position createdAt
                    }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("id", id)
                if (content != null) put("content", content)
                if (type != null) put("type", type)
            }
        )
        return client.decode(client.extractData(response, "updateItem"))
    }

    suspend fun deleteItem(id: String): Boolean {
        val response = client.execute(
            query = """
                mutation DeleteItem(${'$'}id: ID!) { deleteItem(id: ${'$'}id) }
            """.trimIndent(),
            variables = buildJsonObject { put("id", id) }
        )
        return client.decode(client.extractData(response, "deleteItem"))
    }

    suspend fun addTag(listId: String, tagName: String): NoteList {
        val response = client.execute(
            query = """
                mutation AddTag(${'$'}listId: ID!, ${'$'}tagName: String!) {
                    addTagToList(listID: ${'$'}listId, tagName: ${'$'}tagName) { $listFields }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("listId", listId)
                put("tagName", tagName)
            }
        )
        return client.decode(client.extractData(response, "addTagToList"))
    }

    suspend fun removeTag(listId: String, tagName: String): NoteList {
        val response = client.execute(
            query = """
                mutation RemoveTag(${'$'}listId: ID!, ${'$'}tagName: String!) {
                    removeTagFromList(listID: ${'$'}listId, tagName: ${'$'}tagName) { $listFields }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("listId", listId)
                put("tagName", tagName)
            }
        )
        return client.decode(client.extractData(response, "removeTagFromList"))
    }
}
