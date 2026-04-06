package com.example.noteapp.widget

import android.app.PendingIntent
import android.appwidget.AppWidgetManager
import android.appwidget.AppWidgetProvider
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.net.Uri
import android.os.Build
import android.widget.RemoteViews
import com.example.noteapp.MainActivity
import com.example.noteapp.R
import com.example.noteapp.data.GraphQLClient
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.TokenStore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class ListWidgetReceiver : AppWidgetProvider() {

    override fun onUpdate(context: Context, appWidgetManager: AppWidgetManager, appWidgetIds: IntArray) {
        appWidgetIds.forEach { updateWidget(context, appWidgetManager, it) }
    }

    override fun onDeleted(context: Context, appWidgetIds: IntArray) {
        appWidgetIds.forEach { WidgetPreferences.clear(context, it) }
    }

    override fun onReceive(context: Context, intent: Intent) {
        super.onReceive(context, intent)
        if (intent.action != ACTION_DELETE_ITEM) return

        val itemId = intent.getStringExtra(EXTRA_ITEM_ID) ?: return
        val appWidgetId = intent.getIntExtra(
            AppWidgetManager.EXTRA_APPWIDGET_ID,
            AppWidgetManager.INVALID_APPWIDGET_ID,
        )
        if (appWidgetId == AppWidgetManager.INVALID_APPWIDGET_ID) return

        val pending = goAsync()
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val tokenStore = TokenStore(context)
                val client = GraphQLClient(WIDGET_BASE_URL, tokenStore)
                val repo = ListRepository(client)
                repo.deleteItem(itemId)
                val listId = WidgetPreferences.getListId(context, appWidgetId)
                if (listId != null) {
                    val list = repo.getList(listId)
                    if (list != null) {
                        val items = list.items
                            .filter { it.type == "TEXT" }
                            .map { WidgetItem(it.id, it.content) }
                        WidgetPreferences.setData(context, appWidgetId, listId, list.title, items)
                    }
                }
                val manager = AppWidgetManager.getInstance(context)
                manager.notifyAppWidgetViewDataChanged(appWidgetId, R.id.widget_list)
                updateWidget(context, manager, appWidgetId)
            } catch (_: Exception) {
            } finally {
                pending.finish()
            }
        }
    }

    companion object {
        const val ACTION_DELETE_ITEM = "com.example.noteapp.widget.ACTION_DELETE_ITEM"
        const val EXTRA_ITEM_ID = "extra_item_id"

        fun updateWidget(context: Context, appWidgetManager: AppWidgetManager, appWidgetId: Int) {
            val listId = WidgetPreferences.getListId(context, appWidgetId)
            if (listId == null) {
                appWidgetManager.updateAppWidget(
                    appWidgetId,
                    RemoteViews(context.packageName, R.layout.widget_unconfigured),
                )
                return
            }

            val views = RemoteViews(context.packageName, R.layout.widget_list)
            views.setTextViewText(R.id.widget_title, WidgetPreferences.getTitle(context, appWidgetId))
            views.setInt(R.id.widget_add, "setColorFilter", Color.parseColor("#7C4DFF"))

            // Header tap → open app to list
            val openIntent = Intent(context, MainActivity::class.java).apply {
                putExtra("openListId", listId)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            }
            views.setOnClickPendingIntent(
                R.id.widget_header,
                PendingIntent.getActivity(
                    context, appWidgetId, openIntent,
                    PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
                ),
            )

            // Add button tap
            val addIntent = Intent(context, AddItemActivity::class.java).apply {
                putExtra("listId", listId)
            }
            views.setOnClickPendingIntent(
                R.id.widget_add,
                PendingIntent.getActivity(
                    context, appWidgetId + 1000, addIntent,
                    PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
                ),
            )

            // Bind ListView to RemoteViewsService
            val serviceIntent = Intent(context, ListWidgetService::class.java).apply {
                putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, appWidgetId)
                data = Uri.parse(toUri(Intent.URI_INTENT_SCHEME))
            }
            views.setRemoteAdapter(R.id.widget_list, serviceIntent)
            views.setEmptyView(R.id.widget_list, R.id.widget_empty)

            // Delete item template PendingIntent (filled in per-item by the factory)
            val deleteIntent = Intent(context, ListWidgetReceiver::class.java).apply {
                action = ACTION_DELETE_ITEM
                putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, appWidgetId)
            }
            val deleteFlag = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S)
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE
            else
                PendingIntent.FLAG_UPDATE_CURRENT
            views.setPendingIntentTemplate(
                R.id.widget_list,
                PendingIntent.getBroadcast(context, appWidgetId + 2000, deleteIntent, deleteFlag),
            )

            appWidgetManager.updateAppWidget(appWidgetId, views)
        }

        fun refreshAllForList(context: Context, listId: String, title: String, items: List<WidgetItem>) {
            val manager = AppWidgetManager.getInstance(context)
            val ids = manager.getAppWidgetIds(ComponentName(context, ListWidgetReceiver::class.java))
            ids.filter { WidgetPreferences.getListId(context, it) == listId }
                .forEach { id ->
                    WidgetPreferences.setData(context, id, listId, title, items)
                    manager.notifyAppWidgetViewDataChanged(id, R.id.widget_list)
                    updateWidget(context, manager, id)
                }
        }
    }
}
