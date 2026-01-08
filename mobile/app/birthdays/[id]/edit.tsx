import { useState, useEffect } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, ActivityIndicator, ScrollView } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { apiClient } from '@/services/api';

export default function EditBirthdayScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [name, setName] = useState('');
  const [surname, setSurname] = useState('');
  const [dateOfBirth, setDateOfBirth] = useState('');
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
      loadBirthday();
    }
  }, [id]);

  const loadBirthday = async () => {
    try {
      const data = await apiClient.getBirthday(id);
      setName(data.name);
      setSurname(data.surname || '');
      setDateOfBirth(data.dateOfBirth);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to load birthday');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!name.trim()) {
      Alert.alert('Error', 'Please enter a name');
      return;
    }

    if (!dateOfBirth.trim()) {
      Alert.alert('Error', 'Please enter a date of birth');
      return;
    }

    // Validate date format (YYYY-MM-DD)
    const dateRegex = /^\d{4}-\d{2}-\d{2}$/;
    if (!dateRegex.test(dateOfBirth)) {
      Alert.alert('Error', 'Please enter date in YYYY-MM-DD format');
      return;
    }

    setSaving(true);
    try {
      await apiClient.updateBirthday(id, {
        name: name.trim(),
        surname: surname.trim() || undefined,
        dateOfBirth: dateOfBirth.trim(),
      });
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update birthday');
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
      <Text style={[styles.title, { color: textColor }]}>Edit Birthday</Text>
      
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

