import React, { useState } from 'react';
import { View, FlatList, StyleSheet, ActivityIndicator, RefreshControl } from 'react-native';
import { Text, Button, Input, Card, Modal, ListItem, Divider } from '@ui-kitten/components';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Stack, router } from 'expo-router';
import { 
  useTags, 
  useCreateTag, 
  useUpdateTag, 
  useDeleteTag 
} from '@/hooks/useShoppingList';
import { Tag } from '@/lib/models';

export default function TagsScreen() {
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTag, setEditingTag] = useState<Partial<Tag> | null>(null);
  const [tagName, setTagName] = useState('');
  const [error, setError] = useState<string | null>(null);
  
  // Fetch data
  const { 
    data: tags, 
    isLoading, 
    isError,
    refetch,
    isRefetching
  } = useTags();
  
  const createTag = useCreateTag();
  const updateTag = useUpdateTag();
  const deleteTag = useDeleteTag();
  
  const handleAddTag = () => {
    setEditingTag(null);
    setTagName('');
    setError(null);
    setModalVisible(true);
  };
  
  const handleEditTag = (tag: Tag) => {
    setEditingTag(tag);
    setTagName(tag.name);
    setError(null);
    setModalVisible(true);
  };
  
  const handleDeleteTag = (id: string) => {
    deleteTag.mutate(id);
  };
  
  const handleSubmit = () => {
    if (!tagName.trim()) {
      setError('Tag name is required');
      return;
    }
    
    if (editingTag?.id) {
      updateTag.mutate(
        { id: editingTag.id, tag: { name: tagName.trim() } },
        {
          onSuccess: () => {
            setModalVisible(false);
            setEditingTag(null);
            setTagName('');
          }
        }
      );
    } else {
      createTag.mutate(
        { name: tagName.trim() },
        {
          onSuccess: () => {
            setModalVisible(false);
            setTagName('');
          }
        }
      );
    }
  };
  
  const renderTagItem = ({ item }: { item: Tag }) => (
    <ListItem
      title={item.name}
      accessoryRight={(props) => (
        <View style={styles.actionButtons}>
          <Button
            size="small"
            appearance="ghost"
            status="basic"
            onPress={() => handleEditTag(item)}
            {...props}
          >
            Edit
          </Button>
          <Button
            size="small"
            appearance="ghost"
            status="danger"
            onPress={() => handleDeleteTag(item.id)}
            {...props}
          >
            Delete
          </Button>
        </View>
      )}
    />
  );
  
  // Render content based on loading/error state
  if (isLoading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" />
      </View>
    );
  }
  
  if (isError) {
    return (
      <View style={styles.centered}>
        <Text status="danger">Failed to load tags</Text>
        <Button onPress={() => refetch()} style={styles.retryButton}>
          Retry
        </Button>
      </View>
    );
  }
  
  return (
    <SafeAreaView style={styles.container}>
      <Stack.Screen options={{ title: 'Tags' }} />
      
      <View style={styles.header}>
        <Button onPress={handleAddTag}>Add Tag</Button>
      </View>
      
      {tags && tags.length === 0 ? (
        <View style={styles.emptyState}>
          <Text appearance="hint">No tags yet</Text>
          <Button 
            appearance="ghost" 
            status="primary" 
            onPress={handleAddTag}
            style={styles.emptyStateButton}
          >
            Add your first tag
          </Button>
        </View>
      ) : (
        <FlatList
          data={tags}
          keyExtractor={(item) => item.id}
          renderItem={renderTagItem}
          ItemSeparatorComponent={Divider}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
            />
          }
        />
      )}
      
      <Modal
        visible={modalVisible}
        backdropStyle={styles.backdrop}
        onBackdropPress={() => setModalVisible(false)}
      >
        <Card disabled>
          <Text category="h6" style={styles.modalTitle}>
            {editingTag ? 'Edit Tag' : 'Add Tag'}
          </Text>
          
          <Input
            placeholder="Tag name"
            value={tagName}
            onChangeText={setTagName}
            status={error ? 'danger' : 'basic'}
            style={styles.input}
          />
          
          <View style={styles.modalButtons}>
            <Button
              appearance="outline"
              status="basic"
              onPress={() => setModalVisible(false)}
              style={styles.modalButton}
              disabled={createTag.isPending || updateTag.isPending}
            >
              Cancel
            </Button>
            <Button
              onPress={handleSubmit}
              style={styles.modalButton}
              disabled={createTag.isPending || updateTag.isPending}
            >
              {createTag.isPending || updateTag.isPending ? 'Saving...' : 'Save'}
            </Button>
          </View>
        </Card>
      </Modal>
      
      <Button
        appearance="ghost"
        status="basic"
        onPress={() => router.back()}
        style={styles.backButton}
      >
        Back to Shopping List
      </Button>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#EEE',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  retryButton: {
    marginTop: 16,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 16,
  },
  emptyStateButton: {
    marginTop: 8,
  },
  backdrop: {
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
  },
  modalTitle: {
    marginBottom: 16,
  },
  input: {
    marginBottom: 16,
  },
  modalButtons: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
  },
  modalButton: {
    marginLeft: 8,
  },
  actionButtons: {
    flexDirection: 'row',
  },
  backButton: {
    margin: 16,
  },
});