import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, Alert, TextInput } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { useColor } from '@/hooks/useColor';
import { apiClient } from '@/services/api';

interface ListItem {
  id: string;
  content: string;
  completed: boolean;
  order: number;
}

interface List {
  id: string;
  title: string;
  description?: string;
  completed: boolean;
  dueDate?: string;
  dueTime?: string;
  items: ListItem[];
}

export default function ListDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [list, setList] = useState<List | null>(null);
  const [loading, setLoading] = useState(true);
  const [newItemText, setNewItemText] = useState('');
  const [addingItem, setAddingItem] = useState(false);
  const [editing, setEditing] = useState(false);
  const router = useRouter();

  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');
  const destructiveColor = useColor({}, 'destructive');
  const inputColor = useColor({}, 'input');
  const borderColor = useColor({}, 'border');

  useEffect(() => {
    if (id) {
      loadList();
    }
  }, [id]);

  const loadList = async () => {
    try {
      const data = await apiClient.getList(id);
      setList(data);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to load list');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const handleAddItem = async () => {
    if (!newItemText.trim() || !list) return;

    setAddingItem(true);
    try {
      await apiClient.createListItem(list.id, { content: newItemText.trim() });
      setNewItemText('');
      await loadList();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to add item');
    } finally {
      setAddingItem(false);
    }
  };

  const handleToggleItem = async (itemId: string, completed: boolean) => {
    if (!list) return;

    try {
      await apiClient.updateListItem(list.id, itemId, { completed: !completed });
      await loadList();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update item');
    }
  };

  const handleDeleteItem = async (itemId: string) => {
    if (!list) return;

    Alert.alert(
      'Delete Item',
      'Are you sure you want to delete this item?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await apiClient.deleteListItem(list.id, itemId);
              await loadList();
            } catch (error: any) {
              Alert.alert('Error', error.message || 'Failed to delete item');
            }
          },
        },
      ]
    );
  };

  const handleDeleteList = async () => {
    if (!list) return;

    Alert.alert(
      'Delete List',
      'Are you sure you want to delete this list?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await apiClient.deleteList(list.id);
              router.back();
            } catch (error: any) {
              Alert.alert('Error', error.message || 'Failed to delete list');
            }
          },
        },
      ]
    );
  };

  const handleToggleComplete = async () => {
    if (!list) return;

    try {
      await apiClient.updateList(list.id, { completed: !list.completed });
      await loadList();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update list');
    }
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <ActivityIndicator size="large" color={primaryColor} />
      </View>
    );
  }

  if (!list) {
    return null;
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()}>
          <Ionicons name="arrow-back" size={24} color={textColor} />
        </TouchableOpacity>
        <View style={styles.headerActions}>
          <TouchableOpacity onPress={() => router.push(`/lists/${id}/edit`)}>
            <Ionicons name="create-outline" size={24} color={primaryColor} />
          </TouchableOpacity>
          <TouchableOpacity onPress={handleDeleteList} style={{ marginLeft: 15 }}>
            <Ionicons name="trash-outline" size={24} color={destructiveColor} />
          </TouchableOpacity>
        </View>
      </View>

      <View style={[styles.card, { backgroundColor: cardColor }]}>
        <View style={styles.titleRow}>
          <Text style={[styles.title, { color: textColor }]}>{list.title}</Text>
          <TouchableOpacity onPress={handleToggleComplete}>
            <Ionicons
              name={list.completed ? 'checkmark-circle' : 'ellipse-outline'}
              size={28}
              color={list.completed ? primaryColor : textMutedColor}
            />
          </TouchableOpacity>
        </View>

        {list.description && (
          <Text style={[styles.description, { color: textMutedColor }]}>
            {list.description}
          </Text>
        )}

        {(list.dueDate || list.dueTime) && (
          <Text style={[styles.dueDate, { color: textMutedColor }]}>
            Due: {list.dueDate} {list.dueTime}
          </Text>
        )}
      </View>

      <View style={styles.addItemContainer}>
        <TextInput
          style={[styles.addItemInput, { backgroundColor: inputColor, borderColor, color: textColor }]}
          placeholder="Add new item..."
          placeholderTextColor={textMutedColor}
          value={newItemText}
          onChangeText={setNewItemText}
          onSubmitEditing={handleAddItem}
        />
        <TouchableOpacity
          style={[styles.addButton, { backgroundColor: primaryColor }]}
          onPress={handleAddItem}
          disabled={addingItem || !newItemText.trim()}
        >
          {addingItem ? (
            <ActivityIndicator color="#FFFFFF" size="small" />
          ) : (
            <Ionicons name="add" size={24} color="#FFFFFF" />
          )}
        </TouchableOpacity>
      </View>

      <View style={styles.itemsContainer}>
        {list.items && list.items.length > 0 ? (
          list.items
            .sort((a, b) => a.order - b.order)
            .map((item) => (
              <View key={item.id} style={[styles.itemCard, { backgroundColor: cardColor }]}>
                <TouchableOpacity
                  style={styles.itemContent}
                  onPress={() => handleToggleItem(item.id, item.completed)}
                >
                  <Ionicons
                    name={item.completed ? 'checkmark-circle' : 'ellipse-outline'}
                    size={24}
                    color={item.completed ? primaryColor : textMutedColor}
                  />
                  <Text
                    style={[
                      styles.itemText,
                      { color: item.completed ? textMutedColor : textColor },
                      item.completed && styles.itemTextCompleted,
                    ]}
                  >
                    {item.content}
                  </Text>
                </TouchableOpacity>
                <TouchableOpacity onPress={() => handleDeleteItem(item.id)}>
                  <Ionicons name="trash-outline" size={20} color={destructiveColor} />
                </TouchableOpacity>
              </View>
            ))
        ) : (
          <Text style={[styles.emptyText, { color: textMutedColor }]}>
            No items yet. Add one above!
          </Text>
        )}
      </View>
    </ScrollView>
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
  headerActions: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  card: {
    padding: 15,
    borderRadius: 12,
    marginBottom: 20,
  },
  titleRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 10,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    flex: 1,
  },
  description: {
    fontSize: 16,
    marginBottom: 10,
  },
  dueDate: {
    fontSize: 14,
  },
  addItemContainer: {
    flexDirection: 'row',
    marginBottom: 20,
    gap: 10,
  },
  addItemInput: {
    flex: 1,
    height: 50,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 15,
    fontSize: 16,
  },
  addButton: {
    width: 50,
    height: 50,
    borderRadius: 8,
    justifyContent: 'center',
    alignItems: 'center',
  },
  itemsContainer: {
    gap: 10,
  },
  itemCard: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 15,
    borderRadius: 12,
  },
  itemContent: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  itemText: {
    fontSize: 16,
    flex: 1,
  },
  itemTextCompleted: {
    textDecorationLine: 'line-through',
  },
  emptyText: {
    textAlign: 'center',
    marginTop: 20,
    fontSize: 16,
  },
});

