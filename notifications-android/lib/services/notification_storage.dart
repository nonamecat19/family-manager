import 'package:hive_flutter/hive_flutter.dart';
import '../models/notification_item.dart';

class NotificationStorage {
  static const _boxName = 'notifications';
  late Box<NotificationItem> _box;

  Future<void> initialize() async {
    await Hive.initFlutter();
    Hive.registerAdapter(NotificationItemAdapter());
    _box = await Hive.openBox<NotificationItem>(_boxName);
  }

  Future<void> add(NotificationItem item) async {
    await _box.add(item);
  }

  List<NotificationItem> getAll() {
    return _box.values.toList().reversed.toList();
  }

  Future<void> clear() async {
    await _box.clear();
  }

  Future<void> delete(int index) async {
    await _box.deleteAt(index);
  }
}
