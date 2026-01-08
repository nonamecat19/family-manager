import { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator, Alert, TextInput, Modal } from 'react-native';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
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
  const textMutedColor = useColor({}, 'textMuted');
  const inputColor = useColor({}, 'input');
  const borderColor = useColor({}, 'border');
  const { families, activeFamily, setActiveFamily, refreshFamilies } = useFamily();
  const [loading, setLoading] = useState(false);
  const [inviteModalVisible, setInviteModalVisible] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviting, setInviting] = useState(false);
  const router = useRouter();

  const handleSelectFamily = async (family: Family) => {
    try {
      await setActiveFamily(family);
      router.back();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to switch workspace');
    }
  };

  const handleInvite = async () => {
    if (!activeFamily) {
      Alert.alert('Error', 'Please select a workspace first');
      return;
    }

    if (!inviteEmail.trim()) {
      Alert.alert('Error', 'Please enter an email address');
      return;
    }

    // Basic email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(inviteEmail.trim())) {
      Alert.alert('Error', 'Please enter a valid email address');
      return;
    }

    setInviting(true);
    try {
      await apiClient.inviteMember(activeFamily.id, { email: inviteEmail.trim() });
      Alert.alert('Success', 'Invitation sent successfully!');
      setInviteEmail('');
      setInviteModalVisible(false);
      await refreshFamilies();
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to send invitation');
    } finally {
      setInviting(false);
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
      <Text style={[styles.familyRole, { color: textMutedColor }]}>
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

      {activeFamily && (
        <TouchableOpacity
          style={[styles.inviteButton, { backgroundColor: primaryColor }]}
          onPress={() => setInviteModalVisible(true)}
        >
          <Ionicons name="person-add" size={20} color="#FFFFFF" />
          <Text style={styles.inviteButtonText}>Invite Member</Text>
        </TouchableOpacity>
      )}

      <Modal
        visible={inviteModalVisible}
        transparent={true}
        animationType="slide"
        onRequestClose={() => setInviteModalVisible(false)}
      >
        <View style={styles.modalOverlay}>
          <View style={[styles.modalContent, { backgroundColor: cardColor }]}>
            <View style={styles.modalHeader}>
              <Text style={[styles.modalTitle, { color: textColor }]}>Invite Member</Text>
              <TouchableOpacity onPress={() => setInviteModalVisible(false)}>
                <Ionicons name="close" size={24} color={textColor} />
              </TouchableOpacity>
            </View>
            
            <Text style={[styles.modalLabel, { color: textMutedColor }]}>
              Enter email address to invite
            </Text>
            
            <TextInput
              style={[styles.modalInput, { backgroundColor: inputColor, borderColor, color: textColor }]}
              placeholder="email@example.com"
              placeholderTextColor={textMutedColor}
              value={inviteEmail}
              onChangeText={setInviteEmail}
              keyboardType="email-address"
              autoCapitalize="none"
              autoCorrect={false}
            />

            <View style={styles.modalButtons}>
              <TouchableOpacity
                style={[styles.modalButton, styles.cancelButton, { borderColor }]}
                onPress={() => {
                  setInviteModalVisible(false);
                  setInviteEmail('');
                }}
              >
                <Text style={[styles.modalButtonText, { color: textColor }]}>Cancel</Text>
              </TouchableOpacity>
              
              <TouchableOpacity
                style={[styles.modalButton, styles.sendButton, { backgroundColor: primaryColor }]}
                onPress={handleInvite}
                disabled={inviting}
              >
                {inviting ? (
                  <ActivityIndicator color="#FFFFFF" />
                ) : (
                  <Text style={styles.modalButtonText}>Send Invitation</Text>
                )}
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>
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
  inviteButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 15,
    borderRadius: 12,
    marginTop: 20,
    gap: 8,
  },
  inviteButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    justifyContent: 'flex-end',
  },
  modalContent: {
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    padding: 20,
    paddingBottom: 40,
  },
  modalHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 20,
  },
  modalTitle: {
    fontSize: 24,
    fontWeight: 'bold',
  },
  modalLabel: {
    fontSize: 14,
    marginBottom: 10,
  },
  modalInput: {
    height: 50,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 15,
    marginBottom: 20,
    fontSize: 16,
  },
  modalButtons: {
    flexDirection: 'row',
    gap: 10,
  },
  modalButton: {
    flex: 1,
    height: 50,
    borderRadius: 8,
    justifyContent: 'center',
    alignItems: 'center',
  },
  cancelButton: {
    borderWidth: 1,
  },
  sendButton: {
    // backgroundColor set inline
  },
  modalButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});

