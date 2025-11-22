import { View, Text, StyleSheet, TouchableOpacity, Alert } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useAuth } from '@/contexts/AuthContext';
import { useFamily } from '@/contexts/FamilyContext';

export default function SettingsScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const primaryColor = useColor({}, 'primary');
  const cardColor = useColor({}, 'card');
  const textMutedColor = useColor({}, 'textMuted');
  const destructiveColor = useColor({}, 'destructive');
  const { user, logout } = useAuth();
  const { activeFamily } = useFamily();
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

  return (
    <View style={[styles.container, { backgroundColor }]}>
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
    </View>
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
});
