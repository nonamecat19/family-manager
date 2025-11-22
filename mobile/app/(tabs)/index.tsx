import { View, Text, StyleSheet, TouchableOpacity, FlatList, ActivityIndicator } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { useAuth } from '@/contexts/AuthContext';

export default function HomeScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');
  const { activeFamily, families, setActiveFamily, loading } = useFamily();
  const { user } = useAuth();
  const router = useRouter();

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <ActivityIndicator size="large" color={primaryColor} />
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor }]}>
      <Text style={[styles.title, { color: textColor }]}>Welcome, {user?.name || user?.email}!</Text>
      
      {activeFamily && (
        <View style={[styles.activeFamilyCard, { backgroundColor: cardColor }]}>
          <Text style={[styles.familyName, { color: textColor }]}>{activeFamily.name}</Text>
          <Text style={[styles.familyRole, { color: textMutedColor }]}>
            {activeFamily.role === 'owner' ? 'Owner' : 'Member'}
          </Text>
        </View>
      )}

      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor }]}
        onPress={() => router.push('/families')}
      >
        <Text style={styles.buttonText}>Manage Workspaces</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor, marginTop: 10 }]}
        onPress={() => router.push('/lists')}
      >
        <Text style={styles.buttonText}>View Lists</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor, marginTop: 10 }]}
        onPress={() => router.push('/notes')}
      >
        <Text style={styles.buttonText}>View Notes</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor, marginTop: 10 }]}
        onPress={() => router.push('/birthdays')}
      >
        <Text style={styles.buttonText}>View Birthdays</Text>
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
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 20,
  },
  activeFamilyCard: {
    padding: 15,
    borderRadius: 12,
    marginBottom: 20,
  },
  familyName: {
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 5,
  },
  familyRole: {
    fontSize: 14,
  },
  button: {
    padding: 15,
    borderRadius: 8,
    alignItems: 'center',
  },
  buttonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});
