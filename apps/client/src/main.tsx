import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './app/globals.css'
import Providers from './app/Providers.tsx'
import Router from './app/Router.tsx'
import './app/pwa.ts'

// biome-ignore lint/style/noNonNullAssertion: react-docs
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Providers>
      <Router />
    </Providers>
  </StrictMode>,
)
