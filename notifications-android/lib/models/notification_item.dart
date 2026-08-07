import 'package:hive/hive.dart';

part 'notification_item.g.dart';

@HiveType(typeId: 0)
class NotificationItem extends HiveObject {
  @HiveField(0)
  final String package;

  @HiveField(1)
  final String title;

  @HiveField(2)
  final String text;

  @HiveField(3)
  final DateTime timestamp;

  NotificationItem({
    required this.package,
    required this.title,
    required this.text,
    required this.timestamp,
  });

  factory NotificationItem.fromMap(Map<dynamic, dynamic> map) {
    return NotificationItem(
      package: map['package'] as String? ?? '',
      title: map['title'] as String? ?? '',
      text: map['text'] as String? ?? '',
      timestamp: DateTime.fromMillisecondsSinceEpoch(
        (map['timestamp'] as num?)?.toInt() ??
            DateTime.now().millisecondsSinceEpoch,
      ),
    );
  }
}
