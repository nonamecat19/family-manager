import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createItem,
  createList,
  deleteItem,
  deleteList,
  fetchItems,
  fetchListById,
  fetchLists,
} from './api'
import type {
  CreateItemDto,
  CreateListDto,
  DeleteItemDto,
  DeleteListDto,
} from './types'

export const queryKeys = {
  lists: ['lists'],
  items: ['items'],
}

export const useShoppingListById = (id: number) => {
  return useQuery({
    queryKey: [...queryKeys.lists, id],
    queryFn: () => fetchListById(id),
  })
}

export const useShoppingLists = () => {
  return useQuery({
    queryKey: queryKeys.lists,
    queryFn: fetchLists,
  })
}

export const useCreateList = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateListDto) => createList(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.lists })
    },
  })
}

export const useDeleteList = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: DeleteListDto) => deleteList(data),
    // biome-ignore lint/suspicious/noExplicitAny: TBD
    onSuccess: (data: any) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.lists })
      if (data.withItems) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.items })
      }
    },
  })
}

export const useShoppingItems = () => {
  return useQuery({
    queryKey: queryKeys.items,
    queryFn: fetchItems,
  })
}

export const useCreateItem = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateItemDto) => createItem(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.items })
      void queryClient.invalidateQueries({ queryKey: queryKeys.lists })
    },
  })
}

export const useDeleteItem = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: DeleteItemDto) => deleteItem(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.items })
      void queryClient.invalidateQueries({ queryKey: queryKeys.lists })
    },
  })
}
