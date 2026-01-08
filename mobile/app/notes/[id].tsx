import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, Alert, Linking } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { useColor } from '@/hooks/useColor';
import { apiClient } from '@/services/api';

interface Note {
  id: string;
  title: string;
  contentType: 'text' | 'link' | 'copy_text' | 'file';
  content?: string;
  fileUrl?: string;
}

export default function NoteDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [note, setNote] = useState<Note | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  const backgroundColor = useColor({}, 'background');
  const textColor = useColor({}, 'text');
  const cardColor = useColor({}, 'card');
  const primaryColor = useColor({}, 'primary');
  const textMutedColor = useColor({}, 'textMuted');
  const destructiveColor = useColor({}, 'destructive');

  useEffect(() => {
    if (id) {
      loadNote();
    }
  }, [id]);

  const loadNote = async () => {
    try {
      const data = await apiClient.getNote(id);
      setNote(data);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to load note');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!note) return;

    Alert.alert(
      'Delete Note',
      'Are you sure you want to delete this note?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await apiClient.deleteNote(note.id);
              router.back();
            } catch (error: any) {
              Alert.alert('Error', error.message || 'Failed to delete note');
            }
          },
        },
      ]
    );
  };

  const handleOpenLink = () => {
    if (note?.content && note.contentType === 'link') {
      Linking.openURL(note.content).catch(() => {
        Alert.alert('Error', 'Could not open link');
      });
    }
  };

  const handleCopyText = () => {
    if (note?.content && note.contentType === 'copy_text') {
      // In a real app, you'd use Clipboard from @react-native-clipboard/clipboard
      Alert.alert('Copied', 'Text copied to clipboard');
    }
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <ActivityIndicator size="large" color={primaryColor} />
      </View>
    );
  }

  if (!note) {
    return null;
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()}>
          <Ionicons name="arrow-back" size={24} color={textColor} />
        </TouchableOpacity>
        <View style={styles.headerActions}>
          <TouchableOpacity onPress={() => router.push(`/notes/${id}/edit`)}>
            <Ionicons name="create-outline" size={24} color={primaryColor} />
          </TouchableOpacity>
          <TouchableOpacity onPress={handleDelete} style={{ marginLeft: 15 }}>
            <Ionicons name="trash-outline" size={24} color={destructiveColor} />
          </TouchableOpacity>
        </View>
      </View>

      <View style={[styles.card, { backgroundColor: cardColor }]}>
        <Text style={[styles.title, { color: textColor }]}>{note.title}</Text>
        <View style={styles.typeBadge}>
          <Text style={[styles.typeText, { color: textMutedColor }]}>
            {note.contentType.replace('_', ' ').toUpperCase()}
          </Text>
        </View>
      </View>

      <View style={[styles.contentCard, { backgroundColor: cardColor }]}>
        {note.contentType === 'link' && note.content ? (
          <TouchableOpacity onPress={handleOpenLink} style={styles.linkContainer}>
            <Ionicons name="link" size={24} color={primaryColor} />
            <Text style={[styles.linkText, { color: primaryColor }]}>{note.content}</Text>
            <Ionicons name="open-outline" size={20} color={primaryColor} />
          </TouchableOpacity>
        ) : note.contentType === 'copy_text' && note.content ? (
          <TouchableOpacity onPress={handleCopyText} style={styles.copyContainer}>
            <Text style={[styles.contentText, { color: textColor }]}>{note.content}</Text>
            <View style={styles.copyButton}>
              <Ionicons name="copy-outline" size={20} color={primaryColor} />
              <Text style={[styles.copyButtonText, { color: primaryColor }]}>Tap to copy</Text>
            </View>
          </TouchableOpacity>
        ) : note.contentType === 'file' && note.fileUrl ? (
          <TouchableOpacity
            onPress={() => Linking.openURL(note.fileUrl!)}
            style={styles.fileContainer}
          >
            <Ionicons name="document" size={48} color={primaryColor} />
            <Text style={[styles.fileText, { color: primaryColor }]}>Open File</Text>
          </TouchableOpacity>
        ) : note.content ? (
          <Text style={[styles.contentText, { color: textColor }]}>{note.content}</Text>
        ) : (
          <Text style={[styles.emptyText, { color: textMutedColor }]}>No content</Text>
        )}
      </View>
    </ScrollView>
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
  headerActions: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  card: {
    padding: 15,
    borderRadius: 12,
    marginBottom: 20,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 10,
  },
  typeBadge: {
    alignSelf: 'flex-start',
  },
  typeText: {
    fontSize: 12,
    textTransform: 'uppercase',
    fontWeight: '600',
  },
  contentCard: {
    padding: 15,
    borderRadius: 12,
    minHeight: 200,
  },
  contentText: {
    fontSize: 16,
    lineHeight: 24,
  },
  linkContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    padding: 15,
    borderRadius: 8,
  },
  linkText: {
    flex: 1,
    fontSize: 16,
  },
  copyContainer: {
    gap: 15,
  },
  copyButton: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginTop: 10,
  },
  copyButtonText: {
    fontSize: 14,
    fontWeight: '600',
  },
  fileContainer: {
    alignItems: 'center',
    justifyContent: 'center',
    padding: 40,
    gap: 15,
  },
  fileText: {
    fontSize: 16,
    fontWeight: '600',
  },
  emptyText: {
    fontSize: 16,
    textAlign: 'center',
    marginTop: 50,
  },
});

