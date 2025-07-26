import axios from 'axios'
import type {
  CreateItemDto,
  CreateListDto,
  DeleteItemDto,
  DeleteListDto,
  ShoppingItem,
  ShoppingList,
} from './types'

const financeApi = axios.create({
  baseURL: import.meta.env.VITE_SERVICE_FINANCE,
})

export const fetchListById = async (id: number) => {
  return financeApi.get<ShoppingList>(`/lists/${id}`).then(({ data }) => data)
}

export const fetchLists = async () => {
  return financeApi.get<ShoppingList[]>(`/lists`).then(({ data }) => data)
}

export const createList = async (data: CreateListDto) => {
  return financeApi.post<ShoppingList>(`/lists`, data).then(({ data }) => data)
}

export const deleteList = async (data: DeleteListDto) => {
  return financeApi
    .delete<ShoppingList>(`/lists`, { data })
    .then(({ data }) => data)
}

export const fetchItems = async () => {
  return financeApi.get<ShoppingItem[]>(`/items`).then(({ data }) => data)
}

export const createItem = async (data: CreateItemDto) => {
  return financeApi.post<ShoppingItem>(`/items`, data).then(({ data }) => data)
}

export const deleteItem = async (data: DeleteItemDto) => {
  return financeApi
    .delete<ShoppingItem>(`/items`, { data })
    .then(({ data }) => data)
}
