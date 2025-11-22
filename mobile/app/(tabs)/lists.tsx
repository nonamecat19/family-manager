import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator, RefreshControl } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

interface List {
  id: string;
  title: string;
  description?: string;
  completed: boolean;
  dueDate?: string;
  dueTime?: string;
  folder?: { id: string; name: string };
  assignedUser?: { id: string; name: string };
}

export default function ListsScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const { activeFamily } = useFamily();
  const [lists, setLists] = useState<List[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const router = useRouter();

  useEffect(() => {
    if (activeFamily) {
      loadLists();
    }
  }, [activeFamily]);

  const loadLists = async () => {
    if (!activeFamily) return;
    
    try {
      const data = await apiClient.getLists({ familyId: activeFamily.id });
      setLists(data);
    } catch (error) {
      console.error('Failed to load lists:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const onRefresh = () => {
    setRefreshing(true);
    loadLists();
  };

  const renderList = ({ item }: { item: List }) => (
    <TouchableOpacity
      style={[styles.listCard, { backgroundColor: cardColor }]}
      onPress={() => router.push(`/lists/${item.id}`)}
    >
      <Text style={[styles.listTitle, { color: textColor }]}>{item.title}</Text>
      {item.description && (
        <Text style={[styles.listDescription, { color: useColor({}, 'textMuted') }]}>
          {item.description}
        </Text>
      )}
      {item.dueDate && (
        <Text style={[styles.listDueDate, { color: useColor({}, 'textMuted') }]}>
          Due: {item.dueDate}
        </Text>
      )}
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
        <Text style={[styles.title, { color: textColor }]}>Lists</Text>
        <TouchableOpacity
          style={[styles.addButton, { backgroundColor: primaryColor }]}
          onPress={() => router.push('/lists/create')}
        >
          <Text style={styles.addButtonText}>+</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={lists}
        renderItem={renderList}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={[styles.emptyText, { color: useColor({}, 'textMuted') }]}>
            No lists yet. Create one to get started!
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
  listCard: {
    padding: 15,
    borderRadius: 12,
  },
  listTitle: {
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 5,
  },
  listDescription: {
    fontSize: 14,
    marginBottom: 5,
  },
  listDueDate: {
    fontSize: 12,
  },
  emptyText: {
    textAlign: 'center',
    marginTop: 50,
    fontSize: 16,
  },
});
