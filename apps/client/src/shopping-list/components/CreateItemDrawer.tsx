import { type FormEvent, type PropsWithChildren, useState } from 'react'
import { Button } from '@/shared/components/ui/button.tsx'
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from '@/shared/components/ui/drawer.tsx'
import { useCreateItem } from '@/shopping-list/hooks.ts'

export default function CreateItemDrawer({
  children,
  listId,
}: PropsWithChildren<{ listId: number }>) {
  const [isOpened, setIsOpened] = useState(false)

  const [newItemName, setNewItemName] = useState('')
  const [newItemDescription, setNewItemDescription] = useState('')
  const createItem = useCreateItem()

  const handleCreateItem = (e: FormEvent) => {
    e.preventDefault()
    if (!newItemName.trim()) return

    createItem.mutate(
      {
        name: newItemName,
        descriptions: newItemDescription || undefined,
        listId,
      },
      {
        onSuccess: () => {
          setNewItemName('')
          setNewItemDescription('')
          setIsOpened(false)
        },
      },
    )
  }

  return (
    <Drawer open={isOpened} onOpenChange={setIsOpened}>
      <DrawerTrigger asChild>{children}</DrawerTrigger>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>New item</DrawerTitle>
        </DrawerHeader>
        <form
          onSubmit={handleCreateItem}
          className="space-y-4 p-4 border rounded-lg"
        >
          <div>
            <label htmlFor="name" className="block text-sm font-medium mb-1">
              List Name
            </label>
            <input
              id="name"
              type="text"
              value={newItemName}
              onChange={(e) => setNewItemName(e.target.value)}
              className="w-full p-2 border rounded-md"
              placeholder="Enter item name"
              required
            />
          </div>
          <div>
            <label
              htmlFor="description"
              className="block text-sm font-medium mb-1"
            >
              Description (optional)
            </label>
            <textarea
              id="description"
              value={newItemDescription}
              onChange={(e) => setNewItemDescription(e.target.value)}
              className="w-full p-2 border rounded-md"
              placeholder="Enter description"
              rows={3}
            />
          </div>

          <DrawerFooter>
            <Button type="submit" disabled={createItem.isPending}>
              {createItem.isPending ? 'Creating...' : 'Create Item'}
            </Button>
            <DrawerClose asChild>
              <Button variant="outline">Cancel</Button>
            </DrawerClose>
          </DrawerFooter>
        </form>
      </DrawerContent>
    </Drawer>
  )
}
