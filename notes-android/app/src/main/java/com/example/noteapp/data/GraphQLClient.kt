package com.example.noteapp.data

import android.content.Context
import android.net.Uri
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.contentType
import io.ktor.serialization.kotlinx.json.json
import kotlinx.coroutines.flow.firstOrNull
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.put

class GraphQLClient(
    private val baseUrl: String,
    private val tokenStore: TokenStore,
) {
    @PublishedApi
    internal val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    private val client = HttpClient(OkHttp) {
        install(ContentNegotiation) {
            json(this@GraphQLClient.json)
        }
    }

    suspend fun execute(
        query: String,
        variables: JsonObject? = null,
    ): JsonObject {
        val response = doRequest(query, variables)

        // Auto-refresh on auth error
        if (isAuthError(response)) {
            if (tryRefresh()) {
                return doRequest(query, variables)
            }
        }

        return response
    }

    private suspend fun doRequest(query: String, variables: JsonObject?): JsonObject {
        val body = buildJsonObject {
            put("query", query)
            if (variables != null) put("variables", variables)
        }
        val token = tokenStore.accessToken.firstOrNull()
        val response = client.post("$baseUrl/graphql") {
            contentType(ContentType.Application.Json)
            setBody(body)
            if (token != null) {
                header("Authorization", "Bearer $token")
            }
        }
        return response.body<JsonObject>()
    }

    private fun isAuthError(response: JsonObject): Boolean {
        val errors = response["errors"] as? kotlinx.serialization.json.JsonArray ?: return false
        if (errors.isEmpty()) return false
        val msg = errors[0].jsonObject["message"]
        if (msg is JsonPrimitive) {
            val text = msg.content.lowercase()
            return text.contains("unauthorized") || text.contains("unauthenticated")
        }
        return false
    }

    private suspend fun tryRefresh(): Boolean {
        return try {
            val refreshToken = tokenStore.refreshToken.firstOrNull() ?: return false
            val body = buildJsonObject {
                put("query", """
                    mutation RefreshToken(${'$'}token: String!) {
                        refreshToken(token: ${'$'}token) {
                            accessToken
                            refreshToken
                        }
                    }
                """.trimIndent())
                put("variables", buildJsonObject { put("token", refreshToken) })
            }
            val response = client.post("$baseUrl/graphql") {
                contentType(ContentType.Application.Json)
                setBody(body)
            }
            val obj = response.body<JsonObject>()
            val data = obj["data"]?.jsonObject ?: return false
            val payload = data["refreshToken"]?.jsonObject ?: return false
            val newAccess = (payload["accessToken"] as? JsonPrimitive)?.content ?: return false
            val newRefresh = (payload["refreshToken"] as? JsonPrimitive)?.content ?: return false
            tokenStore.save(newAccess, newRefresh)
            true
        } catch (_: Exception) {
            false
        }
    }

    suspend fun executeUpload(context: Context, uri: Uri): String {
        val token = tokenStore.accessToken.firstOrNull()

        val operations = """{"query":"mutation UploadImage(${'$'}file: Upload!) { uploadImage(file: ${'$'}file) }","variables":{"file":null}}"""
        val map = """{"0":["variables.file"]}"""

        val inputStream = context.contentResolver.openInputStream(uri)
            ?: throw Exception("Cannot open image")
        val bytes = inputStream.use { it.readBytes() }
        val mimeType = context.contentResolver.getType(uri) ?: "image/jpeg"

        val boundary = "----GraphQLBoundary${System.currentTimeMillis()}"
        val body = buildMultipartBody(boundary, operations, map, bytes, mimeType)

        val response = client.post("$baseUrl/graphql") {
            header("Content-Type", "multipart/form-data; boundary=$boundary")
            if (token != null) header("Authorization", "Bearer $token")
            setBody(body)
        }

        val text = response.bodyAsText()
        val obj = json.parseToJsonElement(text).jsonObject
        val data = obj["data"]?.jsonObject
            ?: throw GraphQLException(extractErrors(obj))
        return (data["uploadImage"] as? JsonPrimitive)?.content
            ?: throw GraphQLException("uploadImage returned null")
    }

    private fun buildMultipartBody(
        boundary: String,
        operations: String,
        map: String,
        fileBytes: ByteArray,
        mimeType: String,
    ): ByteArray {
        val sb = StringBuilder()
        fun part(name: String, content: String, contentType: String = "application/json") {
            sb.append("--$boundary\r\n")
            sb.append("Content-Disposition: form-data; name=\"$name\"\r\n")
            sb.append("Content-Type: $contentType\r\n\r\n")
            sb.append(content)
            sb.append("\r\n")
        }
        part("operations", operations)
        part("map", map)

        // File part
        sb.append("--$boundary\r\n")
        sb.append("Content-Disposition: form-data; name=\"0\"; filename=\"upload\"\r\n")
        sb.append("Content-Type: $mimeType\r\n\r\n")

        val prefix = sb.toString().toByteArray(Charsets.UTF_8)
        val suffix = "\r\n--$boundary--\r\n".toByteArray(Charsets.UTF_8)

        return prefix + fileBytes + suffix
    }

    inline fun <reified T> decode(element: JsonElement): T {
        return json.decodeFromJsonElement(kotlinx.serialization.serializer(), element)
    }

    fun extractData(response: JsonObject, field: String): JsonElement {
        val data = response["data"]?.jsonObject
            ?: throw GraphQLException(extractErrors(response))
        return data[field] ?: throw GraphQLException("Field '$field' not found in response")
    }

    private fun extractErrors(response: JsonObject): String {
        val errors = response["errors"]
        if (errors != null) {
            val arr = errors as? kotlinx.serialization.json.JsonArray
            if (arr != null && arr.isNotEmpty()) {
                val msg = arr[0].jsonObject["message"]
                if (msg is JsonPrimitive) return msg.content
            }
        }
        return "Unknown GraphQL error"
    }
}

class GraphQLException(message: String) : Exception(message)
