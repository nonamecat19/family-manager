package com.example.noteapp.widget

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject

data class WidgetItem(val id: String, val content: String)

object WidgetPreferences {
    private fun prefs(context: Context) =
        context.getSharedPreferences("widget_prefs", Context.MODE_PRIVATE)

    fun setData(context: Context, appWidgetId: Int, listId: String, title: String, items: List<WidgetItem>) {
        val itemsJson = JSONArray().apply {
            items.forEach { item ->
                put(JSONObject().apply {
                    put("id", item.id)
                    put("content", item.content)
                })
            }
        }.toString()
        prefs(context).edit()
            .putString("listId_$appWidgetId", listId)
            .putString("title_$appWidgetId", title)
            .putString("items_$appWidgetId", itemsJson)
            .commit()
    }

    fun getListId(context: Context, appWidgetId: Int): String? =
        prefs(context).getString("listId_$appWidgetId", null)

    fun getTitle(context: Context, appWidgetId: Int): String =
        prefs(context).getString("title_$appWidgetId", "") ?: ""

    fun getItems(context: Context, appWidgetId: Int): List<WidgetItem> {
        val json = prefs(context).getString("items_$appWidgetId", null) ?: return emptyList()
        return try {
            val arr = JSONArray(json)
            (0 until arr.length()).map {
                val obj = arr.getJSONObject(it)
                WidgetItem(obj.getString("id"), obj.getString("content"))
            }
        } catch (_: Exception) {
            emptyList()
        }
    }

    fun clear(context: Context, appWidgetId: Int) {
        prefs(context).edit()
            .remove("listId_$appWidgetId")
            .remove("title_$appWidgetId")
            .remove("items_$appWidgetId")
            .commit()
    }
}
