const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:3000';

// Log the API URL for debugging
console.log('API Base URL:', API_BASE_URL);

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
    console.log('ApiClient initialized with base URL:', this.baseUrl);
  }

  private getToken(): string | null {
    return localStorage.getItem('auth_token');
  }

  private setToken(token: string): void {
    localStorage.setItem('auth_token', token);
  }

  private removeToken(): void {
    localStorage.removeItem('auth_token');
  }

  async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = this.getToken();
    const url = `${this.baseUrl}${endpoint}`;

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    try {
      console.log('Making request to:', url);
      const response = await fetch(url, {
        ...options,
        headers,
      });
      console.log('Response status:', response.status);

      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Request failed' }));
        throw new Error(error.error || `HTTP error! status: ${response.status}`);
      }

      return response.json();
    } catch (error: any) {
      if (error instanceof TypeError && error.message === 'Failed to fetch') {
        throw new Error(`Cannot connect to server at ${this.baseUrl}. Make sure the backend is running.`);
      }
      throw error;
    }
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
    this.setToken(response.token);
    return response;
  }

  async register(email: string, password: string, name?: string) {
    const response = await this.post<{ token: string; user: any; defaultFamily: any }>('/auth/register', {
      email,
      password,
      name,
    });
    this.setToken(response.token);
    return response;
  }

  async logout() {
    this.removeToken();
  }

  async refreshToken() {
    const response = await this.post<{ token: string }>('/auth/refresh');
    this.setToken(response.token);
    return response;
  }
}

export const apiClient = new ApiClient(API_BASE_URL);

