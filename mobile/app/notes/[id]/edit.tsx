import { useState, useEffect } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, ActivityIndicator, ScrollView } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { useColor } from '@/hooks/useColor';
import { apiClient } from '@/services/api';

export default function EditNoteScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [title, setTitle] = useState('');
  const [contentType, setContentType] = useState<'text' | 'link' | 'copy_text' | 'file'>('text');
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const router = useRouter();

  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const inputColor = useColor({}, 'input');
  const borderColor = useColor({}, 'border');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');
  const cardColor = useColor({}, 'card');

  useEffect(() => {
    if (id) {
      loadNote();
    }
  }, [id]);

  const loadNote = async () => {
    try {
      const data = await apiClient.getNote(id);
      setTitle(data.title);
      setContentType(data.contentType);
      setContent(data.content || '');
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to load note');
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
      await apiClient.updateNote(id, {
        title: title.trim(),
        contentType,
        content: content.trim() || undefined,
      });
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update note');
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
      <Text style={[styles.title, { color: textColor }]}>Edit Note</Text>
      
      <Text style={[styles.label, { color: textColor }]}>Title *</Text>
      <TextInput
        style={[styles.input, { backgroundColor: inputColor, borderColor, color: textColor }]}
        placeholder="Note title"
        placeholderTextColor={textMutedColor}
        value={title}
        onChangeText={setTitle}
        autoCapitalize="sentences"
      />

      <Text style={[styles.label, { color: textColor }]}>Content Type</Text>
      <View style={styles.typeContainer}>
        {(['text', 'link', 'copy_text', 'file'] as const).map((type) => (
          <TouchableOpacity
            key={type}
            style={[
              styles.typeButton,
              { backgroundColor: contentType === type ? primaryColor : cardColor, borderColor },
              contentType === type && { borderWidth: 2 },
            ]}
            onPress={() => setContentType(type)}
          >
            <Text
              style={[
                styles.typeButtonText,
                { color: contentType === type ? '#FFFFFF' : textColor },
              ]}
            >
              {type.replace('_', ' ').toUpperCase()}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {contentType !== 'file' && (
        <>
          <Text style={[styles.label, { color: textColor }]}>Content</Text>
          <TextInput
            style={[styles.textArea, { backgroundColor: inputColor, borderColor, color: textColor }]}
            placeholder={
              contentType === 'link' ? 'https://example.com' : 'Enter your content here...'
            }
            placeholderTextColor={textMutedColor}
            value={content}
            onChangeText={setContent}
            multiline
            numberOfLines={8}
            textAlignVertical="top"
            keyboardType={contentType === 'link' ? 'url' : 'default'}
            autoCapitalize={contentType === 'link' ? 'none' : 'sentences'}
          />
        </>
      )}

      {contentType === 'file' && (
        <View style={[styles.infoBox, { backgroundColor: cardColor }]}>
          <Text style={[styles.infoText, { color: textMutedColor }]}>
            File upload is managed separately. Use the detail view to upload files.
          </Text>
        </View>
      )}
      
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
    minHeight: 200,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 15,
    paddingVertical: 12,
    fontSize: 16,
  },
  typeContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 10,
    marginBottom: 10,
  },
  typeButton: {
    paddingHorizontal: 15,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
  },
  typeButtonText: {
    fontSize: 12,
    fontWeight: '600',
  },
  infoBox: {
    padding: 15,
    borderRadius: 8,
    marginTop: 10,
  },
  infoText: {
    fontSize: 14,
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

