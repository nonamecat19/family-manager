import React from 'react';
import { View, StyleSheet, TouchableOpacity } from 'react-native';
import { Text, CheckBox, Button } from '@ui-kitten/components';
import { ListItem as ListItemType } from '@/lib/models';

interface ListItemProps {
  item: ListItemType;
  categoryName?: string;
  tags?: string[];
  onToggleComplete: (id: string, completed: boolean) => void;
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
}

export const ListItem = ({
  item,
  categoryName,
  tags,
  onToggleComplete,
  onEdit,
  onDelete,
}: ListItemProps) => {
  return (
    <View style={styles.container}>
      <View style={styles.row}>
        <CheckBox
          checked={item.completed}
          onChange={(checked) => onToggleComplete(item.id, checked)}
          style={styles.checkbox}
        />
        <View style={styles.content}>
          <Text
            style={[
              styles.name,
              item.completed && styles.completedText,
            ]}
          >
            {item.title}
          </Text>
          {categoryName && (
            <Text appearance="hint" style={styles.category}>
              {categoryName}
            </Text>
          )}
          {tags && tags.length > 0 && (
            <View style={styles.tagsContainer}>
              {tags.map((tag) => (
                <View key={tag} style={styles.tag}>
                  <Text style={styles.tagText}>{tag}</Text>
                </View>
              ))}
            </View>
          )}
        </View>
      </View>
      <View style={styles.actions}>
        <Button
          size="small"
          appearance="ghost"
          status="basic"
          onPress={() => onEdit(item.id)}
        >
          Edit
        </Button>
        <Button
          size="small"
          appearance="ghost"
          status="danger"
          onPress={() => onDelete(item.id)}
        >
          Delete
        </Button>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#EEE',
  },
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  checkbox: {
    marginRight: 12,
  },
  content: {
    flex: 1,
  },
  name: {
    fontSize: 16,
    fontWeight: '500',
  },
  completedText: {
    textDecorationLine: 'line-through',
    color: '#888',
  },
  category: {
    fontSize: 14,
    marginTop: 4,
  },
  tagsContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    marginTop: 8,
  },
  tag: {
    backgroundColor: '#E8F0FE',
    borderRadius: 12,
    paddingHorizontal: 8,
    paddingVertical: 2,
    marginRight: 8,
    marginBottom: 4,
  },
  tagText: {
    fontSize: 12,
    color: '#1A73E8',
  },
  actions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    marginTop: 8,
  },
});