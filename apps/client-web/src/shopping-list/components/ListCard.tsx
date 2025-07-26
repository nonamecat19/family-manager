import React from 'react'
import { Card, CardTitle } from '../../shared/components/ui/card'
import type { ShoppingList } from '../types'

interface ShoppingListCardProps {
  list: ShoppingList
}

export const ListCard: React.FC<ShoppingListCardProps> = ({ list }) => {
  const completedItemsCount = list.items.filter(
    (item) => item.isCompleted,
  ).length
  const totalItemsCount = list.items.length

  return (
    <Card className="w-full">
      <div className="py-2 px-5 flex justify-between items-center w-full">
        <CardTitle className="text-lg">{list.name}</CardTitle>
        <p className="text-sm text-muted-foreground">
          {completedItemsCount} of {totalItemsCount} items completed
        </p>
      </div>
    </Card>
  )
}
