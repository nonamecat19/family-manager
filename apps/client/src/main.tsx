import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './main.css'
import { createBrowserRouter, RouterProvider } from 'react-router'
import { registerSW } from 'virtual:pwa-register';
import App from './App.tsx'

registerSW({
  onNeedRefresh() {
    // Show a toast or button to refresh
  },
  onOfflineReady() {
    console.log('App is ready to work offline');
  },
});

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
