import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { useColor } from '@/hooks/useColor';
import { apiClient } from '@/services/api';

interface Birthday {
  id: string;
  name: string;
  surname?: string;
  dateOfBirth: string;
  daysUntil: number;
  nextBirthday: string;
  user?: { id: string; name: string };
}

export default function BirthdayDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [birthday, setBirthday] = useState<Birthday | null>(null);
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
      loadBirthday();
    }
  }, [id]);

  const loadBirthday = async () => {
    try {
      const data = await apiClient.getBirthday(id);
      setBirthday(data);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to load birthday');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!birthday) return;

    Alert.alert(
      'Delete Birthday',
      'Are you sure you want to delete this birthday?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await apiClient.deleteBirthday(birthday.id);
              router.back();
            } catch (error: any) {
              Alert.alert('Error', error.message || 'Failed to delete birthday');
            }
          },
        },
      ]
    );
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor }]}>
        <ActivityIndicator size="large" color={primaryColor} />
      </View>
    );
  }

  if (!birthday) {
    return null;
  }

  const birthDate = new Date(birthday.dateOfBirth);
  const formattedDate = birthDate.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  return (
    <ScrollView style={[styles.container, { backgroundColor }]}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()}>
          <Ionicons name="arrow-back" size={24} color={textColor} />
        </TouchableOpacity>
        <View style={styles.headerActions}>
          <TouchableOpacity onPress={() => router.push(`/birthdays/${id}/edit`)}>
            <Ionicons name="create-outline" size={24} color={primaryColor} />
          </TouchableOpacity>
          <TouchableOpacity onPress={handleDelete} style={{ marginLeft: 15 }}>
            <Ionicons name="trash-outline" size={24} color={destructiveColor} />
          </TouchableOpacity>
        </View>
      </View>

      <View style={[styles.card, { backgroundColor: cardColor }]}>
        <View style={styles.nameContainer}>
          <Ionicons name="gift" size={48} color={primaryColor} />
          <Text style={[styles.name, { color: textColor }]}>
            {birthday.name} {birthday.surname || ''}
          </Text>
        </View>

        <View style={styles.infoSection}>
          <Text style={[styles.label, { color: textMutedColor }]}>Date of Birth</Text>
          <Text style={[styles.value, { color: textColor }]}>{formattedDate}</Text>
        </View>

        <View style={styles.infoSection}>
          <Text style={[styles.label, { color: textMutedColor }]}>Next Birthday</Text>
          <Text style={[styles.value, { color: textColor }]}>
            {new Date(birthday.nextBirthday).toLocaleDateString('en-US', {
              month: 'long',
              day: 'numeric',
              year: 'numeric',
            })}
          </Text>
        </View>

        <View style={[styles.daysContainer, { backgroundColor: primaryColor + '20' }]}>
          <Text style={[styles.daysLabel, { color: textMutedColor }]}>Days Until Birthday</Text>
          <Text style={[styles.daysValue, { color: primaryColor }]}>
            {birthday.daysUntil === 0 ? 'Today!' : `${birthday.daysUntil} days`}
          </Text>
        </View>
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
    padding: 20,
    borderRadius: 12,
    alignItems: 'center',
  },
  nameContainer: {
    alignItems: 'center',
    marginBottom: 30,
  },
  name: {
    fontSize: 28,
    fontWeight: 'bold',
    marginTop: 15,
    textAlign: 'center',
  },
  infoSection: {
    width: '100%',
    marginBottom: 20,
  },
  label: {
    fontSize: 14,
    marginBottom: 5,
  },
  value: {
    fontSize: 18,
    fontWeight: '600',
  },
  daysContainer: {
    width: '100%',
    padding: 20,
    borderRadius: 12,
    alignItems: 'center',
    marginTop: 10,
  },
  daysLabel: {
    fontSize: 14,
    marginBottom: 8,
  },
  daysValue: {
    fontSize: 32,
    fontWeight: 'bold',
  },
});

