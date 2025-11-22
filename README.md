# Family Manager App

A comprehensive family management application with mobile (Expo + BNA UI) and desktop (Tauri + shadcn/ui) clients, featuring real-time synchronization.

## Features

### Core Features

1. **Families/Workspaces**
   - Multi-tenant support with workspace switching
   - Create, manage, and invite members to families
   - Private workspaces for individual users

2. **Lists**
   - Folder-based organization
   - Customizable folders (name, icon, color, tags)
   - Assign lists to family members
   - Due dates and times
   - List items with ordering

3. **Notes**
   - Folder-based organization
   - Multiple content types: text, link, copy text, file
   - File upload via Vercel Blob
   - Customizable folders

4. **Birthdays**
   - Track family member birthdays
   - Days until calculation
   - Sorted by day
   - Upcoming birthdays view

### Technical Features

- **Real-time Sync**: WebSocket-based updates across devices
- **Authentication**: JWT-based auth with email/password
- **File Storage**: Vercel Blob integration
- **Database**: PostgreSQL with Drizzle ORM
- **Mobile**: Expo + BNA UI (iOS-style design)
- **Desktop**: Tauri v2 + React + TypeScript

## Getting Started

### Prerequisites

- Node.js 18+
- PostgreSQL database
- Vercel Blob account (for file storage)
- Rust (for Tauri desktop app)

### Backend Setup

1. Navigate to backend directory:
```bash
cd backend
```

2. Install dependencies:
```bash
npm install
```

3. Create `.env` file:
```env
POSTGRES_URL=postgresql://user:password@localhost:5432/family_manager
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRES_IN=7d
BLOB_READ_WRITE_TOKEN=your-vercel-blob-token
PORT=3000
NODE_ENV=development
CORS_ORIGIN=http://localhost:8081,http://localhost:1420
```

4. Run database migrations:
```bash
npm run db:generate
npm run db:push
```

5. Start the server:
```bash
npm run dev
```

### Mobile App Setup

1. Navigate to mobile directory:
```bash
cd mobile
```

2. Install dependencies:
```bash
npm install
```

3. Create `.env` file (optional):
```env
EXPO_PUBLIC_API_URL=http://localhost:3000
EXPO_PUBLIC_WS_URL=ws://localhost:3000
```

4. Start the app:
```bash
npm start
```

### Desktop App Setup

1. Navigate to desktop directory:
```bash
cd desktop
```

2. Install dependencies:
```bash
npm install
```

3. Initialize Tauri (if not already done):
```bash
npx tauri init
```

4. Run development server:
```bash
npm run dev
```

5. Run Tauri app:
```bash
npm run tauri dev
```

## API Endpoints

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - Login
- `POST /auth/refresh` - Refresh token
- `GET /auth/me` - Get current user

### Families
- `GET /families` - List user's families
- `POST /families` - Create family
- `GET /families/:id` - Get family details
- `PUT /families/:id` - Update family
- `DELETE /families/:id` - Delete family
- `POST /families/:id/invite` - Invite member
- `POST /families/:id/join` - Join family
- `POST /families/switch` - Switch active workspace

### Lists
- `GET /lists` - Get lists (filtered by family, folder, assigned_to)
- `POST /lists` - Create list
- `GET /lists/:id` - Get list with items
- `PUT /lists/:id` - Update list
- `DELETE /lists/:id` - Delete list
- `POST /lists/:id/items` - Add list item
- `PUT /lists/:id/items/:itemId` - Update list item
- `DELETE /lists/:id/items/:itemId` - Delete list item

### Notes
- `GET /notes` - Get notes (filtered by family, folder)
- `POST /notes` - Create note
- `GET /notes/:id` - Get note details
- `PUT /notes/:id` - Update note
- `DELETE /notes/:id` - Delete note
- `POST /notes/:id/upload` - Upload file

### Birthdays
- `GET /birthdays` - Get birthdays (family context, sorted by day)
- `POST /birthdays` - Create birthday entry
- `GET /birthdays/:id` - Get birthday details
- `PUT /birthdays/:id` - Update birthday
- `DELETE /birthdays/:id` - Delete birthday
- `GET /birthdays/upcoming` - Get upcoming birthdays

### Folders
- `GET /folders` - Get folders (type: list/note, family context)
- `POST /folders` - Create folder
- `GET /folders/:id` - Get folder with children
- `PUT /folders/:id` - Update folder
- `DELETE /folders/:id` - Delete folder

## WebSocket Events

- `list_updated` - List created/updated/deleted
- `note_updated` - Note created/updated/deleted
- `birthday_updated` - Birthday created/updated/deleted
- `family_updated` - Family created/updated/deleted or member changes
- `folder_updated` - Folder created/updated/deleted

## Development Notes

- The mobile app uses BNA UI components (iOS-style design)
- The desktop app structure is set up but needs shadcn/ui integration
- Real-time sync is implemented via WebSocket rooms per family
- File uploads use Vercel Blob storage
- All API endpoints require authentication except register/login

## Next Steps

1. Complete desktop app UI with shadcn/ui components
2. Add more detailed error handling
3. Implement offline support with local caching
4. Add unit and integration tests
5. Set up CI/CD pipeline
6. Add comprehensive documentation

## License

MIT

