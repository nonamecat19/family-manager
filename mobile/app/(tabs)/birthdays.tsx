import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator, RefreshControl } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

interface Birthday {
  id: string;
  name: string;
  surname?: string;
  dateOfBirth: string;
  daysUntil: number;
  nextBirthday: string;
  user?: { id: string; name: string };
}

export default function BirthdaysScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');
  const { activeFamily } = useFamily();
  const [birthdays, setBirthdays] = useState<Birthday[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const router = useRouter();

  useEffect(() => {
    if (activeFamily) {
      loadBirthdays();
    }
  }, [activeFamily]);

  const loadBirthdays = async () => {
    if (!activeFamily) return;
    
    try {
      const data = await apiClient.getBirthdays({ familyId: activeFamily.id });
      setBirthdays(data);
    } catch (error) {
      console.error('Failed to load birthdays:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const onRefresh = () => {
    setRefreshing(true);
    loadBirthdays();
  };

  const renderBirthday = ({ item }: { item: Birthday }) => (
    <TouchableOpacity
      style={[styles.birthdayCard, { backgroundColor: cardColor }]}
      onPress={() => router.push(`/birthdays/${item.id}`)}
    >
      <View style={styles.birthdayHeader}>
        <Text style={[styles.birthdayName, { color: textColor }]}>
          {item.name} {item.surname || ''}
        </Text>
        <Text style={[styles.daysUntil, { color: primaryColor }]}>
          {item.daysUntil === 0 ? 'Today!' : `${item.daysUntil} days`}
        </Text>
      </View>
      <Text style={[styles.birthdayDate, { color: textMutedColor }]}>
        {new Date(item.dateOfBirth).toLocaleDateString('en-US', { month: 'long', day: 'numeric' })}
      </Text>
    </TouchableOpacity>
  );

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <ActivityIndicator size="large" color={primaryColor} />
      </View>
    );
  }

  if (!activeFamily) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <Text style={[styles.emptyText, { color: textColor }]}>
          Please select a workspace
        </Text>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor }]}>
      <View style={styles.header}>
        <Text style={[styles.title, { color: textColor }]}>Birthdays</Text>
        <TouchableOpacity
          style={[styles.addButton, { backgroundColor: primaryColor }]}
          onPress={() => router.push('/birthdays/create')}
        >
          <Text style={styles.addButtonText}>+</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={birthdays}
        renderItem={renderBirthday}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={[styles.emptyText, { color: textMutedColor }]}>
            No birthdays yet. Add one to get started!
          </Text>
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 20,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
  },
  addButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    justifyContent: 'center',
    alignItems: 'center',
  },
  addButtonText: {
    color: '#FFFFFF',
    fontSize: 24,
    fontWeight: 'bold',
  },
  list: {
    gap: 15,
  },
  birthdayCard: {
    padding: 15,
    borderRadius: 12,
  },
  birthdayHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 5,
  },
  birthdayName: {
    fontSize: 18,
    fontWeight: '600',
    flex: 1,
  },
  daysUntil: {
    fontSize: 14,
    fontWeight: '600',
  },
  birthdayDate: {
    fontSize: 14,
  },
  emptyText: {
    textAlign: 'center',
    marginTop: 50,
    fontSize: 16,
  },
});
