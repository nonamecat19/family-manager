package com.example.noteapp.data

import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

class AuthRepository(
    private val client: GraphQLClient,
    private val tokenStore: TokenStore,
) {
    suspend fun register(email: String, username: String, password: String): AuthPayload {
        val response = client.execute(
            query = """
                mutation Register(${'$'}email: String!, ${'$'}username: String!, ${'$'}password: String!) {
                    register(input: { email: ${'$'}email, username: ${'$'}username, password: ${'$'}password }) {
                        user { id email username }
                        accessToken
                        refreshToken
                    }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("email", email)
                put("username", username)
                put("password", password)
            }
        )
        val data = client.extractData(response, "register")
        val payload: AuthPayload = client.decode(data)
        tokenStore.save(payload.accessToken, payload.refreshToken)
        return payload
    }

    suspend fun login(email: String, password: String): AuthPayload {
        val response = client.execute(
            query = """
                mutation Login(${'$'}email: String!, ${'$'}password: String!) {
                    login(input: { email: ${'$'}email, password: ${'$'}password }) {
                        user { id email username }
                        accessToken
                        refreshToken
                    }
                }
            """.trimIndent(),
            variables = buildJsonObject {
                put("email", email)
                put("password", password)
            }
        )
        val data = client.extractData(response, "login")
        val payload: AuthPayload = client.decode(data)
        tokenStore.save(payload.accessToken, payload.refreshToken)
        return payload
    }

    suspend fun me(): User {
        val response = client.execute(
            query = """
                query { me { id email username createdAt } }
            """.trimIndent()
        )
        return client.decode(client.extractData(response, "me"))
    }

    suspend fun logout() {
        tokenStore.clear()
    }
}
