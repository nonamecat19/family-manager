import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './main.css'
import { createBrowserRouter, RouterProvider } from 'react-router'
import App from './App.tsx'

const router = createBrowserRouter([
  {
    path: '/',
    Component: App,
  },
])

// biome-ignore lint/style/noNonNullAssertion: react-docs
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
