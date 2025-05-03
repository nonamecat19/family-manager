import React, { useState } from 'react';
import { View, FlatList, StyleSheet, ActivityIndicator, RefreshControl } from 'react-native';
import { Text, Button, Input, Card, Modal, ListItem, Divider } from '@ui-kitten/components';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Stack, router } from 'expo-router';
import { 
  useCategories, 
  useCreateCategory, 
  useUpdateCategory, 
  useDeleteCategory 
} from '@/hooks/useShoppingList';
import { Category } from '@/lib/models';

export default function CategoriesScreen() {
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Partial<Category> | null>(null);
  const [categoryName, setCategoryName] = useState('');
  const [error, setError] = useState<string | null>(null);
  
  // Fetch data
  const { 
    data: categories, 
    isLoading, 
    isError,
    refetch,
    isRefetching
  } = useCategories();
  
  // Mutations
  const createCategory = useCreateCategory();
  const updateCategory = useUpdateCategory();
  const deleteCategory = useDeleteCategory();
  
  // Handle category actions
  const handleAddCategory = () => {
    setEditingCategory(null);
    setCategoryName('');
    setError(null);
    setModalVisible(true);
  };
  
  const handleEditCategory = (category: Category) => {
    setEditingCategory(category);
    setCategoryName(category.name);
    setError(null);
    setModalVisible(true);
  };
  
  const handleDeleteCategory = (id: string) => {
    deleteCategory.mutate(id);
  };
  
  const handleSubmit = () => {
    if (!categoryName.trim()) {
      setError('Category name is required');
      return;
    }
    
    if (editingCategory?.id) {
      updateCategory.mutate(
        { id: editingCategory.id, category: { name: categoryName.trim() } },
        {
          onSuccess: () => {
            setModalVisible(false);
            setEditingCategory(null);
            setCategoryName('');
          }
        }
      );
    } else {
      createCategory.mutate(
        { name: categoryName.trim() },
        {
          onSuccess: () => {
            setModalVisible(false);
            setCategoryName('');
          }
        }
      );
    }
  };
  
  const renderCategoryItem = ({ item }: { item: Category }) => (
    <ListItem
      title={item.name}
      accessoryRight={(props) => (
        <View style={styles.actionButtons}>
          <Button
            size="small"
            appearance="ghost"
            status="basic"
            onPress={() => handleEditCategory(item)}
            {...props}
          >
            Edit
          </Button>
          <Button
            size="small"
            appearance="ghost"
            status="danger"
            onPress={() => handleDeleteCategory(item.id)}
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
        <Text status="danger">Failed to load categories</Text>
        <Button onPress={() => refetch()} style={styles.retryButton}>
          Retry
        </Button>
      </View>
    );
  }
  
  return (
    <SafeAreaView style={styles.container}>
      <Stack.Screen options={{ title: 'Categories' }} />
      
      <View style={styles.header}>
        <Button onPress={handleAddCategory}>Add Category</Button>
      </View>
      
      {categories && categories.length === 0 ? (
        <View style={styles.emptyState}>
          <Text appearance="hint">No categories yet</Text>
          <Button 
            appearance="ghost" 
            status="primary" 
            onPress={handleAddCategory}
            style={styles.emptyStateButton}
          >
            Add your first category
          </Button>
        </View>
      ) : (
        <FlatList
          data={categories}
          keyExtractor={(item) => item.id}
          renderItem={renderCategoryItem}
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
            {editingCategory ? 'Edit Category' : 'Add Category'}
          </Text>
          
          <Input
            placeholder="Category name"
            value={categoryName}
            onChangeText={setCategoryName}
            status={error ? 'danger' : 'basic'}
            style={styles.input}
          />
          
          <View style={styles.modalButtons}>
            <Button
              appearance="outline"
              status="basic"
              onPress={() => setModalVisible(false)}
              style={styles.modalButton}
              disabled={createCategory.isPending || updateCategory.isPending}
            >
              Cancel
            </Button>
            <Button
              onPress={handleSubmit}
              style={styles.modalButton}
              disabled={createCategory.isPending || updateCategory.isPending}
            >
              {createCategory.isPending || updateCategory.isPending ? 'Saving...' : 'Save'}
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