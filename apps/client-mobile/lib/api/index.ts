import { ListItem, Category, Tag } from '../models';
import axios, {AxiosResponse} from "axios";

const API_URL = 'http://192.168.31.117/shopping-service';

const axiosInstance = axios.create({
  baseURL: API_URL,
});

axiosInstance.interceptors.response.use(
    (response) => response,
    (error) => {
      console.error('API Error:', {
        url: error.config?.url,
        status: error.response?.status,
        data: error.response?.data,
      });
      throw new Error(error.response?.data?.message || 'API request failed');
    }
);

async function apiRequest<T>(
    url: string,
    method: 'GET' | 'POST' | 'PUT' | 'DELETE' = 'GET',
    data?: any
): Promise<T> {
  try {
    const response: AxiosResponse<T> = await axiosInstance({
      method,
      url,
      data
    });

    return response.data;
  } catch (error) {
    throw error;
  }
}


export const listItemsApi = {
  getAll: () => apiRequest<ListItem[]>('/list-items'),
  getById: (id: string) => apiRequest<ListItem>(`/list-items/${id}`),
  create: (item: Omit<ListItem, 'id'>) =>
    apiRequest<ListItem>('/list-items', 'POST', item),
  update: (id: string, item: Partial<Omit<ListItem, 'id'>>) =>
    apiRequest<ListItem>(`/list-items/${id}`, 'PUT', item),
  delete: (id: string) => apiRequest<void>(`/list-items/${id}`, 'DELETE'),
};

export const categoriesApi = {
  getAll: () => apiRequest<Category[]>('/categories'),
  getById: (id: string) => apiRequest<Category>(`/categories/${id}`),
  create: (category: Omit<Category, 'id'>) =>
    apiRequest<Category>('/categories', 'POST', category),
  update: (id: string, category: Partial<Omit<Category, 'id'>>) =>
    apiRequest<Category>(`/categories/${id}`, 'PUT', category),
  delete: (id: string) => apiRequest<void>(`/categories/${id}`, 'DELETE'),
};

// Tags API
export const tagsApi = {
  getAll: () => apiRequest<Tag[]>('/tags'),
  getById: (id: string) => apiRequest<Tag>(`/tags/${id}`),
  create: (tag: Omit<Tag, 'id'>) =>
    apiRequest<Tag>('/tags', 'POST', tag),
  update: (id: string, tag: Partial<Omit<Tag, 'id'>>) =>
    apiRequest<Tag>(`/tags/${id}`, 'PUT', tag),
  delete: (id: string) => apiRequest<void>(`/tags/${id}`, 'DELETE'),
};