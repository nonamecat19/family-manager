import { useState, useEffect } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, ActivityIndicator, ScrollView } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { apiClient } from '@/services/api';

export default function EditListScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [dueTime, setDueTime] = useState('');
  const [completed, setCompleted] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const router = useRouter();

  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const inputColor = useColor({}, 'input');
  const borderColor = useColor({}, 'border');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');

  useEffect(() => {
    if (id) {
      loadList();
    }
  }, [id]);

  const loadList = async () => {
    try {
      const data = await apiClient.getList(id);
      setTitle(data.title);
      setDescription(data.description || '');
      setDueDate(data.dueDate || '');
      setDueTime(data.dueTime || '');
      setCompleted(data.completed || false);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to load list');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!title.trim()) {
      Alert.alert('Error', 'Please enter a title');
      return;
    }

    setSaving(true);
    try {
      await apiClient.updateList(id, {
        title: title.trim(),
        description: description.trim() || undefined,
        dueDate: dueDate || undefined,
        dueTime: dueTime || undefined,
        completed,
      });
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update list');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <ActivityIndicator size="large" color={primaryColor} />
      </View>
    );
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <Text style={[styles.title, { color: textColor }]}>Edit List</Text>
      
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
        style={[styles.checkboxContainer, { borderColor }]}
        onPress={() => setCompleted(!completed)}
      >
        <Text style={[styles.checkboxLabel, { color: textColor }]}>Completed</Text>
        <View style={[styles.checkbox, { backgroundColor: completed ? primaryColor : 'transparent', borderColor }]}>
          {completed && <Text style={styles.checkboxCheck}>✓</Text>}
        </View>
      </TouchableOpacity>
      
      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor }]}
        onPress={handleSave}
        disabled={saving}
      >
        {saving ? (
          <ActivityIndicator color="#FFFFFF" />
        ) : (
          <Text style={styles.buttonText}>Save Changes</Text>
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
  checkboxContainer: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: 15,
    borderWidth: 1,
    borderRadius: 8,
    marginTop: 20,
  },
  checkboxLabel: {
    fontSize: 16,
  },
  checkbox: {
    width: 24,
    height: 24,
    borderWidth: 2,
    borderRadius: 4,
    justifyContent: 'center',
    alignItems: 'center',
  },
  checkboxCheck: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: 'bold',
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

