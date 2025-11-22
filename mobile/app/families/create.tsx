import { useState } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, ActivityIndicator } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

export default function CreateFamilyScreen() {
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const { refreshFamilies } = useFamily();
  const router = useRouter();

  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const inputColor = useColor({}, 'input');
  const borderColor = useColor({}, 'border');
  const primaryColor = useColor({}, 'primary');

  const handleCreate = async () => {
    if (!name.trim()) {
      Alert.alert('Error', 'Please enter a workspace name');
      return;
    }

    setLoading(true);
    try {
      await apiClient.createFamily({ name });
      await refreshFamilies();
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to create workspace');
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={[styles.container, { backgroundColor }]}>
      <Text style={[styles.title, { color: textColor }]}>Create Workspace</Text>
      
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="Workspace name"
        placeholderTextColor={useColor({}, 'textMuted')}
        value={name}
        onChangeText={setName}
        autoCapitalize="words"
      />
      
      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor }]}
        onPress={handleCreate}
        disabled={loading}
      >
        {loading ? (
          <ActivityIndicator color="#FFFFFF" />
        ) : (
          <Text style={styles.buttonText}>Create</Text>
        )}
      </TouchableOpacity>
    </View>
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
  input: {
    height: 50,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 15,
    marginBottom: 20,
    fontSize: 16,
  },
  button: {
    height: 50,
    borderRadius: 8,
    justifyContent: 'center',
    alignItems: 'center',
  },
  buttonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});

