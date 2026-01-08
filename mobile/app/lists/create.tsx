import { useState } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, ActivityIndicator, ScrollView } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

export default function CreateListScreen() {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [dueTime, setDueTime] = useState('');
  const [loading, setLoading] = useState(false);
  const { activeFamily } = useFamily();
  const router = useRouter();

  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const inputColor = useColor({}, 'input');
  const borderColor = useColor({}, 'border');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');

  const handleCreate = async () => {
    if (!title.trim()) {
      Alert.alert('Error', 'Please enter a title');
      return;
    }

    if (!activeFamily) {
      Alert.alert('Error', 'Please select a workspace');
      return;
    }

    setLoading(true);
    try {
      await apiClient.createList({
        title: title.trim(),
        description: description.trim() || undefined,
        dueDate: dueDate || undefined,
        dueTime: dueTime || undefined,
        familyId: activeFamily.id,
      });
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to create list');
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <Text style={[styles.title, { color: textColor }]}>Create List</Text>
      
      <Text style={[styles.label, { color: textColor }]}>Title *</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="List title"
        placeholderTextColor={textMutedColor}
        value={title}
        onChangeText={setTitle}
        autoCapitalize="sentences"
      />

      <Text style={[styles.label, { color: textColor }]}>Description</Text>
      <TextInput
        style={[styles.textArea, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="Optional description"
        placeholderTextColor={textMutedColor}
        value={description}
        onChangeText={setDescription}
        multiline
        numberOfLines={4}
        textAlignVertical="top"
      />

      <Text style={[styles.label, { color: textColor }]}>Due Date (YYYY-MM-DD)</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="2024-12-31"
        placeholderTextColor={textMutedColor}
        value={dueDate}
        onChangeText={setDueDate}
      />

      <Text style={[styles.label, { color: textColor }]}>Due Time (HH:MM)</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="14:30"
        placeholderTextColor={textMutedColor}
        value={dueTime}
        onChangeText={setDueTime}
      />
      
      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor }]}
        onPress={handleCreate}
        disabled={loading}
      >
        {loading ? (
          <ActivityIndicator color="#FFFFFF" />
        ) : (
          <Text style={styles.buttonText}>Create List</Text>
        )}
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    marginBottom: 30,
  },
  label: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 8,
    marginTop: 15,
  },
  input: {
    height: 50,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 15,
    fontSize: 16,
  },
  textArea: {
    minHeight: 100,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 15,
    paddingVertical: 12,
    fontSize: 16,
  },
  button: {
    height: 50,
    borderRadius: 8,
    justifyContent: 'center',
    alignItems: 'center',
    marginTop: 30,
  },
  buttonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});

