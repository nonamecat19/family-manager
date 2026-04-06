package com.example.noteapp.widget

import android.appwidget.AppWidgetManager
import android.content.Context
import android.content.Intent
import android.widget.RemoteViews
import android.widget.RemoteViewsService
import com.example.noteapp.R

class ListWidgetService : RemoteViewsService() {
    override fun onGetViewFactory(intent: Intent): RemoteViewsFactory =
        ListWidgetFactory(applicationContext, intent)
}

private class ListWidgetFactory(
    private val context: Context,
    intent: Intent,
) : RemoteViewsService.RemoteViewsFactory {

    private val appWidgetId = intent.getIntExtra(
        AppWidgetManager.EXTRA_APPWIDGET_ID,
        AppWidgetManager.INVALID_APPWIDGET_ID,
    )
    private var items: List<WidgetItem> = emptyList()

    override fun onCreate() {}
    override fun onDestroy() {}

    override fun onDataSetChanged() {
        items = WidgetPreferences.getItems(context, appWidgetId)
    }

    override fun getCount(): Int = items.size

    override fun getViewAt(position: Int): RemoteViews {
        val item = items.getOrNull(position)
            ?: return RemoteViews(context.packageName, R.layout.widget_item)

        val views = RemoteViews(context.packageName, R.layout.widget_item)
        views.setTextViewText(R.id.item_text, item.content)

        // Fill-in intent merged with the template PendingIntent set in ListWidgetReceiver
        val fillIn = Intent().putExtra(ListWidgetReceiver.EXTRA_ITEM_ID, item.id)
        views.setOnClickFillInIntent(R.id.item_delete, fillIn)

        return views
    }

    override fun getLoadingView(): RemoteViews? = null
    override fun getViewTypeCount(): Int = 1
    override fun getItemId(position: Int): Long =
        items.getOrNull(position)?.id?.hashCode()?.toLong() ?: position.toLong()
    override fun hasStableIds(): Boolean = true
}
