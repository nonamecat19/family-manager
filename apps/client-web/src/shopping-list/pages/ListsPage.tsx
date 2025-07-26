import { Link, Outlet } from 'react-router'
import { Button } from '@/shared/components/ui/button.tsx'
import CreateListDrawer from '@/shopping-list/components/CreateListDrawer.tsx'
import { ListCard } from '@/shopping-list/components/ListCard.tsx'
import { useShoppingLists } from '@/shopping-list/hooks.ts'

function ListsNotFound() {
  return (
    <div className="text-center p-8 border rounded-lg">
      <p className="text-muted-foreground">
        No shopping lists found. Create your first list!
      </p>
    </div>
  )
}

export default function ListsPage() {
  const { data: lists, isLoading, isError } = useShoppingLists()

  if (isLoading) {
    return <div className="text-center p-4">Loading shopping lists...</div>
  }

  if (isError) {
    return (
      <div className="text-center p-4 text-red-500">
        Error loading shopping lists
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center p-2">
        <h1 className="text-3xl font-bold">Lists</h1>

        <CreateListDrawer>
          <Button>
            <div>New List</div>
          </Button>
        </CreateListDrawer>
      </div>

      <div className="p-2">
        {lists && lists.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {lists.map((list) => (
              <Link to={`/lists/${list.id}`} key={list.id}>
                <ListCard list={list} />
              </Link>
            ))}
          </div>
        ) : (
          <ListsNotFound />
        )}
      </div>
      <Outlet />
    </div>
  )
}
