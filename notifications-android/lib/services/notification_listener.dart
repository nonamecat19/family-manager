import 'package:flutter/services.dart';
import '../models/notification_item.dart';

typedef NotificationCallback = void Function(NotificationItem notification);

class NotificationListener {
  static const _channel = MethodChannel('notifications');
  NotificationCallback? _onNotification;

  void setCallback(NotificationCallback callback) {
    _onNotification = callback;
  }

  Future<void> initialize() async {
    _channel.setMethodCallHandler(_handleMethod);
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    if (call.method == 'onNotification' && call.arguments is Map) {
      final data = Map<dynamic, dynamic>.from(call.arguments as Map);
      final item = NotificationItem.fromMap(data);
      _onNotification?.call(item);
    }
  }

  void dispose() {
    _channel.setMethodCallHandler(null);
  }
}
