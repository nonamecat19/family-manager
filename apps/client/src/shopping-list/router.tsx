import type { RouteObject } from 'react-router'
import ItemsPage from '@/shopping-list/pages/ItemsPage.tsx'
import ListItemPage from '@/shopping-list/pages/ListItemPage.tsx'
import ListsPage from '@/shopping-list/pages/ListsPage.tsx'

export const financeRouter: RouteObject[] = [
  {
    path: '/',
    Component: ListsPage,
  },
  {
    path: '/lists',
    children: [
      {
        path: ':id',
        Component: ListItemPage,
      },
    ],
  },
  {
    path: '/items',
    Component: ItemsPage,
  },
]
