import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import { useColor } from '@/hooks/useColor';
import { useNotifications } from '@/contexts/NotificationContext';

export function NotificationBanner() {
  const { notifications, unreadCount, markAsRead, clearNotification } = useNotifications();
  const insets = useSafeAreaInsets();
  const backgroundColor = useColor({}, 'card');
  const textColor = useColor({}, 'text');
  const textMutedColor = useColor({}, 'textMuted');
  const primaryColor = useColor({}, 'primary');
  const borderColor = useColor({}, 'border');

  // Show the most recent unread notification
  const latestUnread = notifications.find((n) => !n.read);

  if (!latestUnread || unreadCount === 0) {
    return null;
  }

  const handleDismiss = () => {
    markAsRead(latestUnread.id);
    clearNotification(latestUnread.id);
  };

  const formatTime = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (seconds < 60) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <View style={[styles.container, { backgroundColor, borderColor, paddingTop: insets.top }]}>
      <View style={styles.content}>
        <View style={styles.iconContainer}>
          <Ionicons
            name={
              latestUnread.type === 'invitation_accepted'
                ? 'checkmark-circle'
                : latestUnread.type === 'member_added'
                ? 'person-add'
                : 'notifications'
            }
            size={20}
            color={primaryColor}
          />
        </View>
        <View style={styles.textContainer}>
          <Text style={[styles.title, { color: textColor }]} numberOfLines={1}>
            {latestUnread.title}
          </Text>
          <Text style={[styles.message, { color: textMutedColor }]} numberOfLines={2}>
            {latestUnread.message}
          </Text>
          <Text style={[styles.time, { color: textMutedColor }]}>
            {formatTime(latestUnread.timestamp)}
          </Text>
        </View>
        <TouchableOpacity onPress={handleDismiss} style={styles.closeButton}>
          <Ionicons name="close" size={20} color={textMutedColor} />
        </TouchableOpacity>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    zIndex: 1000,
    borderBottomWidth: 1,
    paddingHorizontal: 15,
    paddingVertical: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 5,
  },
  content: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  iconContainer: {
    marginRight: 12,
  },
  textContainer: {
    flex: 1,
  },
  title: {
    fontSize: 14,
    fontWeight: '600',
    marginBottom: 2,
  },
  message: {
    fontSize: 12,
    marginBottom: 2,
  },
  time: {
    fontSize: 10,
  },
  closeButton: {
    padding: 4,
  },
});

