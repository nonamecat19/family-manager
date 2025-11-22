import AsyncStorage from '@react-native-async-storage/async-storage';

const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL;

if (!API_BASE_URL) {
  throw new Error('EXPO_PUBLIC_API_URL environment variable is required. Please set it in your eas.json build configuration.');
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    if (!baseUrl) {
      throw new Error('API base URL is required');
    }
    this.baseUrl = baseUrl;
  }

  private async getToken(): Promise<string | null> {
    return AsyncStorage.getItem('auth_token');
  }

  private async setToken(token: string): Promise<void> {
    await AsyncStorage.setItem('auth_token', token);
  }

  private async removeToken(): Promise<void> {
    await AsyncStorage.removeItem('auth_token');
  }

  async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = await this.getToken();
    const url = `${this.baseUrl}${endpoint}`;

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || `HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  async post<T>(endpoint: string, data?: any): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async put<T>(endpoint: string, data?: any): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }

  async login(email: string, password: string) {
    const response = await this.post<{ token: string; user: any }>('/auth/login', {
      email,
      password,
    });
    await this.setToken(response.token);
    return response;
  }

  async register(email: string, password: string, name?: string) {
    const response = await this.post<{ token: string; user: any; defaultFamily: any }>('/auth/register', {
      email,
      password,
      name,
    });
    await this.setToken(response.token);
    return response;
  }

  async logout() {
    await this.removeToken();
  }

  async refreshToken() {
    const response = await this.post<{ token: string }>('/auth/refresh');
    await this.setToken(response.token);
    return response;
  }

  // Families API
  async getFamilies() {
    return this.get<any[]>('/families');
  }

  async getFamily(id: string) {
    return this.get<any>(`/families/${id}`);
  }

  async createFamily(data: { name: string; icon?: string; color?: string }) {
    return this.post<any>('/families', data);
  }

  async updateFamily(id: string, data: { name?: string; icon?: string; color?: string }) {
    return this.put<any>(`/families/${id}`, data);
  }

  async deleteFamily(id: string) {
    return this.delete<any>(`/families/${id}`);
  }

  async inviteMember(familyId: string, data: { email: string }) {
    return this.post<any>(`/families/${familyId}/invite`, data);
  }

  async joinFamily(familyId: string) {
    return this.post<any>(`/families/${familyId}/join`);
  }

  async updateMemberRole(familyId: string, userId: string, data: { role: 'owner' | 'member' }) {
    return this.put<any>(`/families/${familyId}/members/${userId}`, data);
  }

  async removeMember(familyId: string, userId: string) {
    return this.delete<any>(`/families/${familyId}/members/${userId}`);
  }

  async switchFamily(familyId: string) {
    return this.post<any>('/families/switch', { familyId });
  }

  // Lists API
  async getLists(params?: { familyId?: string; folderId?: string; assignedTo?: string }) {
    const queryParams = new URLSearchParams();
    if (params?.familyId) queryParams.append('familyId', params.familyId);
    if (params?.folderId) queryParams.append('folderId', params.folderId);
    if (params?.assignedTo) queryParams.append('assignedTo', params.assignedTo);
    const query = queryParams.toString();
    return this.get<any[]>(`/lists${query ? `?${query}` : ''}`);
  }

  async getList(id: string) {
    return this.get<any>(`/lists/${id}`);
  }

  async createList(data: {
    title: string;
    description?: string;
    folderId?: string;
    assignedTo?: string;
    dueDate?: string;
    dueTime?: string;
    familyId: string;
  }) {
    return this.post<any>('/lists', data);
  }

  async updateList(id: string, data: {
    title?: string;
    description?: string;
    folderId?: string;
    assignedTo?: string;
    dueDate?: string;
    dueTime?: string;
    completed?: boolean;
  }) {
    return this.put<any>(`/lists/${id}`, data);
  }

  async deleteList(id: string) {
    return this.delete<any>(`/lists/${id}`);
  }

  async createListItem(listId: string, data: { content: string; order?: number }) {
    return this.post<any>(`/lists/${listId}/items`, data);
  }

  async updateListItem(listId: string, itemId: string, data: { content?: string; order?: number; completed?: boolean }) {
    return this.put<any>(`/lists/${listId}/items/${itemId}`, data);
  }

  async deleteListItem(listId: string, itemId: string) {
    return this.delete<any>(`/lists/${listId}/items/${itemId}`);
  }

  // Notes API
  async getNotes(params?: { familyId?: string; folderId?: string }) {
    const queryParams = new URLSearchParams();
    if (params?.familyId) queryParams.append('familyId', params.familyId);
    if (params?.folderId) queryParams.append('folderId', params.folderId);
    const query = queryParams.toString();
    return this.get<any[]>(`/notes${query ? `?${query}` : ''}`);
  }

  async getNote(id: string) {
    return this.get<any>(`/notes/${id}`);
  }

  async createNote(data: {
    title: string;
    contentType: 'text' | 'link' | 'copy_text' | 'file';
    content?: string;
    folderId?: string;
    familyId: string;
  }) {
    return this.post<any>('/notes', data);
  }

  async updateNote(id: string, data: {
    title?: string;
    contentType?: 'text' | 'link' | 'copy_text' | 'file';
    content?: string;
    folderId?: string;
  }) {
    return this.put<any>(`/notes/${id}`, data);
  }

  async deleteNote(id: string) {
    return this.delete<any>(`/notes/${id}`);
  }

  async uploadNoteFile(noteId: string, file: { uri: string; type: string; name: string }) {
    const formData = new FormData();
    formData.append('file', {
      uri: file.uri,
      type: file.type,
      name: file.name,
    } as any);

    const token = await this.getToken();
    const url = `${this.baseUrl}/notes/${noteId}/upload`;

    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }));
      throw new Error(error.error || `HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  // Birthdays API
  async getBirthdays(params?: { familyId?: string }) {
    const queryParams = new URLSearchParams();
    if (params?.familyId) queryParams.append('familyId', params.familyId);
    const query = queryParams.toString();
    return this.get<any[]>(`/birthdays${query ? `?${query}` : ''}`);
  }

  async getUpcomingBirthdays(params?: { familyId?: string; limit?: number }) {
    const queryParams = new URLSearchParams();
    if (params?.familyId) queryParams.append('familyId', params.familyId);
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    const query = queryParams.toString();
    return this.get<any[]>(`/birthdays/upcoming${query ? `?${query}` : ''}`);
  }

  async getBirthday(id: string) {
    return this.get<any>(`/birthdays/${id}`);
  }

  async createBirthday(data: {
    name: string;
    surname?: string;
    dateOfBirth: string;
    familyId: string;
    userId?: string;
  }) {
    return this.post<any>('/birthdays', data);
  }

  async updateBirthday(id: string, data: {
    name?: string;
    surname?: string;
    dateOfBirth?: string;
    userId?: string;
  }) {
    return this.put<any>(`/birthdays/${id}`, data);
  }

  async deleteBirthday(id: string) {
    return this.delete<any>(`/birthdays/${id}`);
  }

  // Folders API
  async getFolders(params?: { familyId?: string; type?: 'list' | 'note' }) {
    const queryParams = new URLSearchParams();
    if (params?.familyId) queryParams.append('familyId', params.familyId);
    if (params?.type) queryParams.append('type', params.type);
    const query = queryParams.toString();
    return this.get<any[]>(`/folders${query ? `?${query}` : ''}`);
  }

  async getFolder(id: string) {
    return this.get<any>(`/folders/${id}`);
  }

  async createFolder(data: {
    name: string;
    icon?: string;
    color?: string;
    familyId: string;
    type: 'list' | 'note';
    parentId?: string;
  }) {
    return this.post<any>('/folders', data);
  }

  async updateFolder(id: string, data: {
    name?: string;
    icon?: string;
    color?: string;
    parentId?: string;
  }) {
    return this.put<any>(`/folders/${id}`, data);
  }

  async deleteFolder(id: string) {
    return this.delete<any>(`/folders/${id}`);
  }
}

export const apiClient = new ApiClient(API_BASE_URL);

