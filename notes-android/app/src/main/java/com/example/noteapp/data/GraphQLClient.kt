package com.example.noteapp.data

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
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
