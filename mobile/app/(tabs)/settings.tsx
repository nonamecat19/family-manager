import { View, Text, StyleSheet, TouchableOpacity, Alert, ScrollView, FlatList } from 'react-native';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { useColor } from '@/hooks/useColor';
import { useAuth } from '@/contexts/AuthContext';
import { useFamily } from '@/contexts/FamilyContext';
import { useTheme, ThemeMode } from '@/hooks/useTheme';
import { useNotifications } from '@/contexts/NotificationContext';

export default function SettingsScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const primaryColor = useColor({}, 'primary');
  const cardColor = useColor({}, 'card');
  const textMutedColor = useColor({}, 'textMuted');
  const destructiveColor = useColor({}, 'destructive');
  const borderColor = useColor({}, 'border');
  const { user, logout } = useAuth();
  const { activeFamily } = useFamily();
  const { themeMode, setTheme } = useTheme();
  const { notifications, unreadCount, markAsRead, clearNotification, markAllAsRead } = useNotifications();
  const router = useRouter();

  const handleLogout = async () => {
    Alert.alert(
      'Logout',
      'Are you sure you want to logout?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Logout',
          style: 'destructive',
          onPress: async () => {
            await logout();
            router.replace('/auth/login');
          },
        },
      ]
    );
  };

  const themes: { mode: ThemeMode; label: string; icon: string }[] = [
    { mode: 'auto', label: 'Auto (System)', icon: 'phone-portrait-outline' },
    { mode: 'light', label: 'Light', icon: 'sunny-outline' },
    { mode: 'dark', label: 'Dark', icon: 'moon-outline' },
    { mode: 'blue', label: 'Blue', icon: 'color-palette-outline' },
  ];

  const handleThemeChange = (mode: ThemeMode) => {
    setTheme(mode);
  };

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <Text style={[styles.title, { color: textColor }]}>Settings</Text>
      
      <View style={[styles.section, { backgroundColor: cardColor }]}>
        <Text style={[styles.sectionTitle, { color: textColor }]}>Account</Text>
        <Text style={[styles.info, { color: textMutedColor }]}>
          Email: {user?.email}
        </Text>
        {user?.name && (
          <Text style={[styles.info, { color: textMutedColor }]}>
            Name: {user.name}
          </Text>
        )}
      </View>

      <View style={[styles.section, { backgroundColor: cardColor }]}>
        <View style={styles.sectionHeader}>
          <Text style={[styles.sectionTitle, { color: textColor }]}>Notifications</Text>
          {unreadCount > 0 && (
            <View style={[styles.badge, { backgroundColor: primaryColor }]}>
              <Text style={styles.badgeText}>{unreadCount}</Text>
            </View>
          )}
        </View>
        {notifications.length === 0 ? (
          <Text style={[styles.info, { color: textMutedColor }]}>No notifications</Text>
        ) : (
          <>
            {notifications.slice(0, 3).map((notification) => (
              <View key={notification.id} style={styles.notificationItem}>
                <View style={styles.notificationContent}>
                  <Text
                    style={[
                      styles.notificationTitle,
                      { color: textColor, fontWeight: notification.read ? '400' : '600' },
                    ]}
                  >
                    {notification.title}
                  </Text>
                  <Text style={[styles.notificationMessage, { color: textMutedColor }]} numberOfLines={1}>
                    {notification.message}
                  </Text>
                </View>
                <TouchableOpacity
                  onPress={() => {
                    markAsRead(notification.id);
                    clearNotification(notification.id);
                  }}
                >
                  <Ionicons name="close" size={18} color={textMutedColor} />
                </TouchableOpacity>
              </View>
            ))}
            {notifications.length > 3 && (
              <Text style={[styles.info, { color: textMutedColor, marginTop: 10 }]}>
                +{notifications.length - 3} more notifications
              </Text>
            )}
            {unreadCount > 0 && (
              <TouchableOpacity
                style={[styles.button, { backgroundColor: primaryColor, marginTop: 10 }]}
                onPress={markAllAsRead}
              >
                <Text style={styles.buttonText}>Mark All as Read</Text>
              </TouchableOpacity>
            )}
          </>
        )}
      </View>

      <View style={[styles.section, { backgroundColor: cardColor }]}>
        <Text style={[styles.sectionTitle, { color: textColor }]}>Appearance</Text>
        <Text style={[styles.info, { color: textMutedColor, marginBottom: 15 }]}>
          Choose your preferred theme
        </Text>
        {themes.map((theme) => (
          <TouchableOpacity
            key={theme.mode}
            style={[
              styles.themeOption,
              {
                backgroundColor: themeMode === theme.mode ? primaryColor + '20' : 'transparent',
                borderColor: themeMode === theme.mode ? primaryColor : borderColor,
              },
            ]}
            onPress={() => handleThemeChange(theme.mode)}
          >
            <View style={styles.themeOptionContent}>
              <Ionicons
                name={theme.icon as any}
                size={20}
                color={themeMode === theme.mode ? primaryColor : textMutedColor}
              />
              <Text
                style={[
                  styles.themeOptionText,
                  {
                    color: themeMode === theme.mode ? primaryColor : textColor,
                    fontWeight: themeMode === theme.mode ? '600' : '400',
                  },
                ]}
              >
                {theme.label}
              </Text>
            </View>
            {themeMode === theme.mode && (
              <Ionicons name="checkmark" size={20} color={primaryColor} />
            )}
          </TouchableOpacity>
        ))}
      </View>

      <View style={[styles.section, { backgroundColor: cardColor }]}>
        <Text style={[styles.sectionTitle, { color: textColor }]}>Workspace</Text>
        {activeFamily && (
          <Text style={[styles.info, { color: textMutedColor }]}>
            Active: {activeFamily.name}
          </Text>
        )}
        <TouchableOpacity
          style={[styles.button, { backgroundColor: primaryColor, marginTop: 10 }]}
          onPress={() => router.push('/families')}
        >
          <Text style={styles.buttonText}>Manage Workspaces</Text>
        </TouchableOpacity>
      </View>

      <TouchableOpacity
        style={[styles.logoutButton, { backgroundColor: destructiveColor }]}
        onPress={handleLogout}
      >
        <Text style={styles.logoutButtonText}>Logout</Text>
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    marginBottom: 20,
  },
  section: {
    padding: 15,
    borderRadius: 12,
    marginBottom: 20,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 10,
  },
  info: {
    fontSize: 14,
    marginBottom: 5,
  },
  button: {
    padding: 12,
    borderRadius: 8,
    alignItems: 'center',
  },
  buttonText: {
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: '600',
  },
  logoutButton: {
    padding: 15,
    borderRadius: 8,
    alignItems: 'center',
    marginTop: 20,
  },
  logoutButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
  themeOption: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 12,
    borderRadius: 8,
    marginBottom: 8,
    borderWidth: 1,
  },
  themeOptionContent: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  themeOptionText: {
    fontSize: 16,
  },
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 10,
  },
  badge: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    paddingHorizontal: 6,
    justifyContent: 'center',
    alignItems: 'center',
  },
  badgeText: {
    color: '#FFFFFF',
    fontSize: 12,
    fontWeight: '600',
  },
  notificationItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 10,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(0,0,0,0.05)',
  },
  notificationContent: {
    flex: 1,
    marginRight: 10,
  },
  notificationTitle: {
    fontSize: 14,
    marginBottom: 4,
  },
  notificationMessage: {
    fontSize: 12,
  },
});
