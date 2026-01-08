import { useState } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, ActivityIndicator, ScrollView } from 'react-native';
import { useRouter } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { useFamily } from '@/contexts/FamilyContext';
import { apiClient } from '@/services/api';

export default function CreateBirthdayScreen() {
  const [name, setName] = useState('');
  const [surname, setSurname] = useState('');
  const [dateOfBirth, setDateOfBirth] = useState('');
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
    if (!name.trim()) {
      Alert.alert('Error', 'Please enter a name');
      return;
    }

    if (!dateOfBirth.trim()) {
      Alert.alert('Error', 'Please enter a date of birth');
      return;
    }

    if (!activeFamily) {
      Alert.alert('Error', 'Please select a workspace');
      return;
    }

    // Validate date format (YYYY-MM-DD)
    const dateRegex = /^\d{4}-\d{2}-\d{2}$/;
    if (!dateRegex.test(dateOfBirth)) {
      Alert.alert('Error', 'Please enter date in YYYY-MM-DD format');
      return;
    }

    setLoading(true);
    try {
      await apiClient.createBirthday({
        name: name.trim(),
        surname: surname.trim() || undefined,
        dateOfBirth: dateOfBirth.trim(),
        familyId: activeFamily.id,
      });
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to create birthday');
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <Text style={[styles.title, { color: textColor }]}>Add Birthday</Text>
      
      <Text style={[styles.label, { color: textColor }]}>Name *</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="First name"
        placeholderTextColor={textMutedColor}
        value={name}
        onChangeText={setName}
        autoCapitalize="words"
      />

      <Text style={[styles.label, { color: textColor }]}>Surname</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="Last name (optional)"
        placeholderTextColor={textMutedColor}
        value={surname}
        onChangeText={setSurname}
        autoCapitalize="words"
      />

      <Text style={[styles.label, { color: textColor }]}>Date of Birth *</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="YYYY-MM-DD (e.g., 1990-05-15)"
        placeholderTextColor={textMutedColor}
        value={dateOfBirth}
        onChangeText={setDateOfBirth}
      />
      <Text style={[styles.hint, { color: textMutedColor }]}>
        Format: YYYY-MM-DD
      </Text>
      
      <TouchableOpacity
        style={[styles.button, { backgroundColor: primaryColor }]}
        onPress={handleCreate}
        disabled={loading}
      >
        {loading ? (
          <ActivityIndicator color="#FFFFFF" />
        ) : (
          <Text style={styles.buttonText}>Add Birthday</Text>
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
  hint: {
    fontSize: 12,
    marginTop: 5,
    marginBottom: 10,
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

