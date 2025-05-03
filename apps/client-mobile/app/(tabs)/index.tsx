import React, {useState} from 'react';
import {
  ActivityIndicator,
  FlatList,
  Modal,
  RefreshControl,
  StyleSheet,
  TouchableWithoutFeedback,
  View
} from 'react-native';
import {Button, Text} from '@ui-kitten/components';
import {SafeAreaView} from 'react-native-safe-area-context';
import {router} from 'expo-router';
import {ListItem} from '@/components/shopping-list/ListItem';
import {ItemForm} from '@/components/shopping-list/ItemForm';
import {
  useCategories,
  useCreateListItem,
  useDeleteListItem,
  useListItems,
  useTags,
  useUpdateListItem
} from '@/hooks/useShoppingList';
import {ListItem as ListItemType} from '@/lib/models';

export default function ShoppingListScreen() {
    const [modalVisible, setModalVisible] = useState(false);
    const [editingItem, setEditingItem] = useState<Partial<ListItemType> | null>(null);

    // Fetch data
    const {
        data: items,
        isLoading: isLoadingItems,
        isError: isErrorItems,
        refetch: refetchItems,
        isRefetching: isRefetchingItems
    } = useListItems();

    const {
        data: categories,
        isLoading: isLoadingCategories
    } = useCategories();

    const {
        data: tags,
        isLoading: isLoadingTags
    } = useTags();

    const createItem = useCreateListItem();
    const updateItem = useUpdateListItem();
    const deleteItem = useDeleteListItem();

    const isLoading = isLoadingItems || isLoadingCategories || isLoadingTags;

    const handleToggleComplete = (id: string, completed: boolean) => {
        updateItem.mutate({id, item: {completed}});
    };

    const handleEditItem = (id: string) => {
        const item = items?.find(item => item.id === id);
        if (item) {
            setEditingItem(item);
            setModalVisible(true);
        }
    };

    const handleDeleteItem = (id: string) => {
        deleteItem.mutate(id);
    };

    const handleAddItem = () => {
        setEditingItem(null);
        setModalVisible(true);
    };

    const handleSubmitItem = (values: Omit<ListItemType, 'id'>) => {
      console.log({values})
        if (editingItem?.id) {
            updateItem.mutate(
                {id: editingItem.id, item: values},
                {
                    onSuccess: () => {
                        setModalVisible(false);
                        setEditingItem(null);
                    }
                }
            );
        } else {
            createItem.mutate(values, {
                onSuccess: () => {
                    setModalVisible(false);
                },
                onError(error) {
                  console.error(error.message)
                }
            });
        }
    };

    const getCategoryName = (categoryId?: string) => {
        if (!categoryId || !categories) return undefined;
        const category = categories.find(c => c.id === categoryId);
        return category?.name;
    };

    const getTagNames = (tagIds?: string[]) => {
        if (!tagIds || !tags) return undefined;
        return tagIds
            .map(id => tags.find(t => t.id === id)?.name)
            .filter(Boolean) as string[];
    };

    if (isLoading) {
        return (
            <View style={styles.centered}>
                <ActivityIndicator size="large"/>
            </View>
        );
    }

    if (isErrorItems) {
        return (
            <View style={styles.centered}>
                <Text status="danger">Failed to load shopping list items</Text>
                <Button onPress={() => refetchItems()} style={styles.retryButton}>
                    Retry
                </Button>
            </View>
        );
    }

    return (
        <SafeAreaView style={styles.container}>
            <View style={styles.header}>
                <Text category="h5">Shopping List</Text>
                <View style={styles.headerButtons}>
                    <Button
                        size="small"
                        appearance="ghost"
                        status="basic"
                        onPress={() => router.push('/categories')}
                        style={styles.headerButton}
                    >
                        Categories
                    </Button>
                    <Button
                        size="small"
                        appearance="ghost"
                        status="basic"
                        onPress={() => router.push('/tags')}
                        style={styles.headerButton}
                    >
                        Tags
                    </Button>
                    <Button onPress={handleAddItem}>Add Item</Button>
                </View>
            </View>

            {items && items.length === 0 ? (
                <View style={styles.emptyState}>
                    <Text appearance="hint">Your shopping list is empty</Text>
                    <Button
                        appearance="ghost"
                        status="primary"
                        onPress={handleAddItem}
                        style={styles.emptyStateButton}
                    >
                        Add your first item
                    </Button>
                </View>
            ) : (
                <FlatList
                    data={items}
                    keyExtractor={(item) => item.id}
                    renderItem={({item}) => (
                        <ListItem
                            item={item}
                            categoryName={getCategoryName(item.categoryId)}
                            tags={getTagNames(item.tagIds)}
                            onToggleComplete={handleToggleComplete}
                            onEdit={handleEditItem}
                            onDelete={handleDeleteItem}
                        />
                    )}
                    refreshControl={
                        <RefreshControl
                            refreshing={isRefetchingItems}
                            onRefresh={refetchItems}
                        />
                    }
                />
            )}

            <Modal
                visible={modalVisible}
                transparent={true}
                animationType="fade"
                onRequestClose={() => setModalVisible(false)}
            >
                <TouchableWithoutFeedback onPress={() => setModalVisible(false)}>
                    <View style={styles.modalOverlay}>
                        <TouchableWithoutFeedback>
                            <View style={styles.modalContent}>
                                <ItemForm
                                    initialValues={editingItem || undefined}
                                    categories={categories || []}
                                    tags={tags || []}
                                    onSubmit={handleSubmitItem}
                                    onCancel={() => setModalVisible(false)}
                                    isSubmitting={createItem.isPending || updateItem.isPending}
                                />
                            </View>
                        </TouchableWithoutFeedback>
                    </View>
                </TouchableWithoutFeedback>
            </Modal>

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
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: 16,
        borderBottomWidth: 1,
        borderBottomColor: '#EEE',
    },
    headerButtons: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    headerButton: {
        marginRight: 8,
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
  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  modalContent: {
    backgroundColor: 'white',
    padding: 20,
    borderRadius: 8,
    width: '90%',
    maxWidth: 400,
    elevation: 5, // for Android shadow
    shadowColor: '#000', // for iOS shadow
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.25,
    shadowRadius: 3.84,
  },
});
