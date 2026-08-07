import 'package:flutter/material.dart';
import 'package:android_intent_plus/android_intent.dart';

import 'models/notification_item.dart';
import 'services/notification_listener.dart' as nl;
import 'services/notification_storage.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const MyApp());
}

class MyApp extends StatefulWidget {
  const MyApp({super.key});

  @override
  State<MyApp> createState() => _MyAppState();
}

class _MyAppState extends State<MyApp> {
  final _storage = NotificationStorage();
  final _listener = nl.NotificationListener();
  List<NotificationItem> _notifications = [];
  bool _hasAccess = true;
  bool _isLoading = true;

  static const _filterPackage = 'org.telegram.messenger';

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    await _storage.initialize();
    _listener.setCallback(_onNotificationReceived);
    await _listener.initialize();
    _loadNotifications();
    setState(() {
      _hasAccess = true;
      _isLoading = false;
    });
  }

  void _onNotificationReceived(NotificationItem notification) {
    if (notification.package == _filterPackage || _filterPackage.isEmpty) {
      _storage.add(notification);
      _loadNotifications();
    }
  }

  void _loadNotifications() {
    setState(() {
      _notifications = _storage.getAll();
    });
  }

  Future<void> _openNotificationSettings() async {
    const intent = AndroidIntent(
      action: 'android.settings.ACTION_NOTIFICATION_LISTENER_SETTINGS',
    );
    await intent.launch();
  }

  Future<void> _clearAll() async {
    await _storage.clear();
    _loadNotifications();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Notification Listener',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
        useMaterial3: true,
      ),
      home: Scaffold(
        appBar: AppBar(
          title: const Text('Notifications'),
          actions: [
            IconButton(
              icon: const Icon(Icons.refresh),
              onPressed: _loadNotifications,
            ),
            IconButton(
              icon: const Icon(Icons.delete_sweep),
              onPressed: _clearAll,
            ),
          ],
        ),
        body: _isLoading
            ? const Center(child: CircularProgressIndicator())
            : !_hasAccess
            ? Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(
                      Icons.notifications_off,
                      size: 64,
                      color: Colors.grey,
                    ),
                    const SizedBox(height: 16),
                    const Text(
                      'Notification access required',
                      style: TextStyle(fontSize: 18),
                    ),
                    const SizedBox(height: 24),
                    ElevatedButton(
                      onPressed: _openNotificationSettings,
                      child: const Text('Grant Access'),
                    ),
                  ],
                ),
              )
            : _notifications.isEmpty
            ? const Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(Icons.inbox, size: 64, color: Colors.grey),
                    SizedBox(height: 16),
                    Text('No notifications yet'),
                    SizedBox(height: 8),
                    Text(
                      'Listening for: All apps',
                      style: TextStyle(color: Colors.grey),
                    ),
                  ],
                ),
              )
            : ListView.builder(
                itemCount: _notifications.length,
                itemBuilder: (context, index) {
                  final item = _notifications[index];
                  return ListTile(
                    leading: CircleAvatar(
                      child: Text(
                        item.package.isNotEmpty
                            ? item.package[0].toUpperCase()
                            : '?',
                      ),
                    ),
                    title: Text(
                      item.title.isNotEmpty ? item.title : '(no title)',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      item.text.isNotEmpty ? item.text : '(no text)',
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: Text(
                      _formatTime(item.timestamp),
                      style: const TextStyle(fontSize: 12, color: Colors.grey),
                    ),
                    onLongPress: () => _deleteNotification(index),
                  );
                },
              ),
      ),
    );
  }

  String _formatTime(DateTime time) {
    return '${time.hour.toString().padLeft(2, '0')}:${time.minute.toString().padLeft(2, '0')}';
  }

  void _deleteNotification(int index) {
    _storage.delete(index);
    _loadNotifications();
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Notification deleted'),
        duration: Duration(seconds: 1),
      ),
    );
  }
}
