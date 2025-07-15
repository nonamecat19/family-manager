import { createBrowserRouter, RouterProvider } from 'react-router'
import { financeRouter } from '../shopping-list/router.tsx'

export default function Router() {
  const router = createBrowserRouter([
    {
      path: '/',
      children: financeRouter,
    },
  ], {
    basename: '/client'
  })

  return <RouterProvider router={router} />
}
