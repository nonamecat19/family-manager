import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './main.css'
import App from './App.tsx'
import {createBrowserRouter, RouterProvider} from "react-router";

const router = createBrowserRouter([
    {
        path: "/",
        Component: App,
    },
]);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
      <RouterProvider router={router} />
  </StrictMode>,
)
