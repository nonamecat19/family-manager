export interface ShoppingList {
  id: number
  name: string
  description?: string
  items: ShoppingItem[]
}

export interface ShoppingItem {
  id: number
  name: string
  description?: string
  quantity: number
  isCompleted: boolean
  listId: number
}

export interface CreateListDto {
  name: string
  descriptions?: string
}

export interface DeleteListDto {
  id: number
  withItems?: boolean
}

export interface CreateItemDto {
  name: string
  descriptions?: string
  quantity?: number
  listId: number
}

export interface DeleteItemDto {
  id: number
}
