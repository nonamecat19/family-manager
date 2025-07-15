import { Trash2, X } from 'lucide-react'
import { useNavigate, useParams } from 'react-router'
import { Button } from '@/shared/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/shared/components/ui/card'
import CreateItemDrawer from '@/shopping-list/components/CreateItemDrawer.tsx'
import {
  useDeleteItem,
  useDeleteList,
  useShoppingListById,
} from '@/shopping-list/hooks.ts'

export default function ListItemPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { data, isLoading, error } = useShoppingListById(+id!)
  const { mutateAsync: deleteItem, isPending: isItemDeleting } = useDeleteItem()
  const { mutateAsync: deleteList, isPending: isListDeleting } = useDeleteList()

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error loading shopping list</div>
  if (!data) return <div>Shopping list not found</div>

  // const handleToggleItem = async (itemId: number, isCompleted: boolean) => {
  // const updatedItems = data.items.map((item) =>
  //   item.id === itemId ? { ...item, isCompleted: !isCompleted } : item,
  // )
  // mutate({ ...data, items: updatedItems }, false)
  // }

  const handleDeleteItem = async (id: number) => {
    try {
      await deleteItem({ id })
    } catch (error) {
      console.error('Failed to delete item:', error)
    }
  }

  const handleDeleteList = async () => {
    try {
      await deleteList({ id: +id! })
      navigate('/')
    } catch (error) {
      console.error('Failed to delete list:', error)
    }
  }

  return (
    <div className="h-screen flex flex-col gap-4 p-4">
      <Card className="w-full max-w-3xl mx-auto flex-1">
        <CardHeader>
          <div className="flex justify-between items-center">
            <CardTitle className="text-2xl">{data.name}</CardTitle>
            <Button
              size="sm"
              onClick={handleDeleteList}
              disabled={isListDeleting}
            >
              <Trash2 className="h-4 w-4" color="white" />
            </Button>
          </div>
          {data.description && (
            <CardDescription>{data.description}</CardDescription>
          )}
        </CardHeader>

        <CardContent>
          {data.items.length === 0 ? (
            <div className="text-center py-6 text-muted-foreground">
              No items in this list yet
            </div>
          ) : (
            <div className="space-y-2">
              {data.items.map((item) => (
                <div
                  key={item.id}
                  className="flex items-center justify-between bg-muted/50 p-2 rounded-md"
                >
                  <div className="flex items-center gap-2">
                    {/*<Checkbox*/}
                    {/*  checked={item.isCompleted}*/}
                    {/*  onCheckedChange={() =>*/}
                    {/*    handleToggleItem(item.id, item.isCompleted)*/}
                    {/*  }*/}
                    {/*/>*/}
                    <div className="flex flex-col">
                      <span
                        className={
                          item.isCompleted
                            ? 'line-through text-muted-foreground'
                            : ''
                        }
                      >
                        {item.name} {item.quantity > 1 && `(${item.quantity})`}
                      </span>
                      {item.description && (
                        <span className="text-xs text-muted-foreground">
                          {item.description}
                        </span>
                      )}
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDeleteItem(item.id)}
                    disabled={isItemDeleting}
                  >
                    <X className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>

        {/*<CardFooter className="border-t pt-6">*/}
        {/*  <div className="text-sm text-muted-foreground">*/}
        {/*    Total: {data.items.reduce((acc, item) => acc + item.quantity, 0)}*/}
        {/*  </div>*/}
        {/*</CardFooter>*/}
      </Card>
      <CreateItemDrawer listId={+id!}>
        <Button>Add item</Button>
      </CreateItemDrawer>
    </div>
  )
}
