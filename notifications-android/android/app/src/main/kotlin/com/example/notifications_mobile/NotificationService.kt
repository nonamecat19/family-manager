package com.example.notifications_mobile

import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.os.Build
import io.flutter.plugin.common.MethodChannel

class NotificationService : NotificationListenerService() {

    companion object {
        private const val CHANNEL_NAME = "notifications"
        private const val METHOD_NAME = "onNotification"
        
        private var methodChannel: MethodChannel? = null
        
        fun setMethodChannel(channel: MethodChannel) {
            methodChannel = channel
        }
    }

    override fun onNotificationPosted(sbn: StatusBarNotification?) {
        sbn ?: return

        val packageName = sbn.packageName
        val extras = sbn.notification.extras

        val title = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            extras.getCharSequence(android.app.Notification.EXTRA_TITLE)?.toString()
        } else {
            @Suppress("DEPRECATION")
            extras.getString(android.app.Notification.EXTRA_TITLE)
        }

        val text = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            extras.getCharSequence(android.app.Notification.EXTRA_TEXT)?.toString()
        } else {
            @Suppress("DEPRECATION")
            extras.getString(android.app.Notification.EXTRA_TEXT)
        }

        val data = mapOf(
            "package" to packageName,
            "title" to (title ?: ""),
            "text" to (text ?: ""),
            "timestamp" to sbn.postTime
        )

        methodChannel?.invokeMethod(METHOD_NAME, data)
    }

    override fun onNotificationRemoved(sbn: StatusBarNotification?) {
        sbn ?: return
    }
}
