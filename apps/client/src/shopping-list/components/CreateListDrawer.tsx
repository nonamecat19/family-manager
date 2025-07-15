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
import { useCreateList } from '@/shopping-list/hooks.ts'

export default function CreateListDrawer({ children }: PropsWithChildren) {
  const [isOpened, setIsOpened] = useState(false)

  const [newListName, setNewListName] = useState('')
  const [newListDescription, setNewListDescription] = useState('')
  const createList = useCreateList()

  const handleCreateList = (e: FormEvent) => {
    e.preventDefault()
    if (!newListName.trim()) return

    createList.mutate(
      {
        name: newListName,
        descriptions: newListDescription || undefined,
      },
      {
        onSuccess: () => {
          setNewListName('')
          setNewListDescription('')
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
          <DrawerTitle>New list</DrawerTitle>
        </DrawerHeader>
        <form
          onSubmit={handleCreateList}
          className="space-y-4 p-4 border rounded-lg"
        >
          <div>
            <label htmlFor="name" className="block text-sm font-medium mb-1">
              List Name
            </label>
            <input
              id="name"
              type="text"
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              className="w-full p-2 border rounded-md"
              placeholder="Enter list name"
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
              value={newListDescription}
              onChange={(e) => setNewListDescription(e.target.value)}
              className="w-full p-2 border rounded-md"
              placeholder="Enter description"
              rows={3}
            />
          </div>

          <DrawerFooter>
            <Button type="submit" disabled={createList.isPending}>
              {createList.isPending ? 'Creating...' : 'Create List'}
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
