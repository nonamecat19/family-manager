# Family Manager Desktop

Desktop application built with Tauri v2, React, TypeScript, and shadcn/ui.

## Setup

1. Install dependencies:
```bash
npm install
```

2. Create `.env` file:
```env
VITE_API_URL=http://localhost:3000
VITE_WS_URL=ws://localhost:3000
```

3. Run development server:
```bash
npm run dev
```

4. (Optional) Run with Tauri:
```bash
npm run tauri dev
```

## Features

- Authentication (Login/Register)
- Workspace management
- Lists management
- Notes management
- Birthdays tracking
- Real-time synchronization via WebSocket
- Modern UI with shadcn/ui components

## Project Structure

```
src/
├── components/
│   ├── ui/          # shadcn/ui components
│   └── Layout.tsx   # Main layout component
├── contexts/        # React contexts (Auth, Family)
├── pages/           # Page components
├── services/        # API client, WebSocket
└── lib/             # Utilities
```
