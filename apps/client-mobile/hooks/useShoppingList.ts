import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listItemsApi, categoriesApi, tagsApi } from '@/lib/api';
import { ListItem, Category, Tag } from '@/lib/models';

// Query keys
const QUERY_KEYS = {
  listItems: 'listItems',
  categories: 'categories',
  tags: 'tags',
};

// List Items Hooks
export function useListItems() {
  return useQuery({
    queryKey: [QUERY_KEYS.listItems],
    queryFn: listItemsApi.getAll,
  });
}

export function useListItem(id: string) {
  return useQuery({
    queryKey: [QUERY_KEYS.listItems, id],
    queryFn: () => listItemsApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateListItem() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (item: Omit<ListItem, 'id'>) =>
      listItemsApi.create(item),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.listItems] });
    },
  });
}

export function useUpdateListItem() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, item }: { id: string; item: Partial<Omit<ListItem, 'id'>> }) =>
      listItemsApi.update(id, item),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.listItems] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.listItems, id] });
    },
  });
}

export function useDeleteListItem() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => listItemsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.listItems] });
    },
  });
}

// Categories Hooks
export function useCategories() {
  return useQuery({
    queryKey: [QUERY_KEYS.categories],
    queryFn: categoriesApi.getAll,
  });
}

export function useCategory(id: string) {
  return useQuery({
    queryKey: [QUERY_KEYS.categories, id],
    queryFn: () => categoriesApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateCategory() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (category: Omit<Category, 'id'>) =>
      categoriesApi.create(category),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.categories] });
    },
  });
}

export function useUpdateCategory() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, category }: { id: string; category: Partial<Omit<Category, 'id'>> }) =>
      categoriesApi.update(id, category),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.categories] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.categories, id] });
    },
  });
}

export function useDeleteCategory() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => categoriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.categories] });
    },
  });
}

// Tags Hooks
export function useTags() {
  return useQuery({
    queryKey: [QUERY_KEYS.tags],
    queryFn: tagsApi.getAll,
  });
}

export function useTag(id: string) {
  return useQuery({
    queryKey: [QUERY_KEYS.tags, id],
    queryFn: () => tagsApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateTag() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (tag: Omit<Tag, 'id'>) =>
      tagsApi.create(tag),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.tags] });
    },
  });
}

export function useUpdateTag() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, tag }: { id: string; tag: Partial<Omit<Tag, 'id'>> }) =>
      tagsApi.update(id, tag),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.tags] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.tags, id] });
    },
  });
}

export function useDeleteTag() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => tagsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.tags] });
    },
  });
}