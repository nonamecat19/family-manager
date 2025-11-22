import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator, RefreshControl } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

interface Note {
  id: string;
  title: string;
  contentType: 'text' | 'link' | 'copy_text' | 'file';
  content?: string;
  fileUrl?: string;
  folder?: { id: string; name: string };
}

export default function NotesScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const { activeFamily } = useFamily();
  const [notes, setNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const router = useRouter();

  useEffect(() => {
    if (activeFamily) {
      loadNotes();
    }
  }, [activeFamily]);

  const loadNotes = async () => {
    if (!activeFamily) return;
    
    try {
      const data = await apiClient.getNotes({ familyId: activeFamily.id });
      setNotes(data);
    } catch (error) {
      console.error('Failed to load notes:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const onRefresh = () => {
    setRefreshing(true);
    loadNotes();
  };

  const renderNote = ({ item }: { item: Note }) => (
    <TouchableOpacity
      style={[styles.noteCard, { backgroundColor: cardColor }]}
      onPress={() => router.push(`/notes/${item.id}`)}
    >
      <Text style={[styles.noteTitle, { color: textColor }]}>{item.title}</Text>
      <Text style={[styles.noteType, { color: useColor({}, 'textMuted') }]}>
        {item.contentType}
      </Text>
      {item.content && (
        <Text style={[styles.noteContent, { color: useColor({}, 'textMuted') }]} numberOfLines={2}>
          {item.content}
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
        <Text style={[styles.title, { color: textColor }]}>Notes</Text>
        <TouchableOpacity
          style={[styles.addButton, { backgroundColor: primaryColor }]}
          onPress={() => router.push('/notes/create')}
        >
          <Text style={styles.addButtonText}>+</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={notes}
        renderItem={renderNote}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={[styles.emptyText, { color: useColor({}, 'textMuted') }]}>
            No notes yet. Create one to get started!
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
  noteCard: {
    padding: 15,
    borderRadius: 12,
  },
  noteTitle: {
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 5,
  },
  noteType: {
    fontSize: 12,
    textTransform: 'uppercase',
    marginBottom: 5,
  },
  noteContent: {
    fontSize: 14,
  },
  emptyText: {
    textAlign: 'center',
    marginTop: 50,
    fontSize: 16,
  },
});
