import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

interface Family {
  id: string;
  name: string;
  icon?: string;
  color?: string;
  role: 'owner' | 'member';
  createdAt: string;
}

export default function FamiliesScreen() {
  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const { families, activeFamily, setActiveFamily, refreshFamilies } = useFamily();
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSelectFamily = async (family: Family) => {
    try {
      await setActiveFamily(family);
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to switch workspace');
    }
  };

  const renderFamily = ({ item }: { item: Family }) => (
    <TouchableOpacity
      style={[
        styles.familyCard,
        { backgroundColor: cardColor },
        activeFamily?.id === item.id && { borderWidth: 2, borderColor: primaryColor },
      ]}
      onPress={() => handleSelectFamily(item)}
    >
      <View style={styles.familyHeader}>
        <Text style={[styles.familyName, { color: textColor }]}>{item.name}</Text>
        {activeFamily?.id === item.id && (
          <Text style={[styles.activeBadge, { color: primaryColor }]}>Active</Text>
        )}
      </View>
      <Text style={[styles.familyRole, { color: useColor({}, 'textMuted') }]}>
        {item.role === 'owner' ? 'Owner' : 'Member'}
      </Text>
    </TouchableOpacity>
  );

  return (
    <View style={[styles.container, { backgroundColor }]}>
      <View style={styles.header}>
        <Text style={[styles.title, { color: textColor }]}>Workspaces</Text>
        <TouchableOpacity
          style={[styles.addButton, { backgroundColor: primaryColor }]}
          onPress={() => router.push('/families/create')}
        >
          <Text style={styles.addButtonText}>+</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={families}
        renderItem={renderFamily}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        refreshing={loading}
        onRefresh={refreshFamilies}
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
  familyCard: {
    padding: 15,
    borderRadius: 12,
  },
  familyHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 5,
  },
  familyName: {
    fontSize: 18,
    fontWeight: '600',
    flex: 1,
  },
  activeBadge: {
    fontSize: 12,
    fontWeight: '600',
  },
  familyRole: {
    fontSize: 14,
  },
});

