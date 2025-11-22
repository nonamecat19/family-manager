# Family Manager Backend

A NestJS backend application with Fastify adapter for the Family Manager application.

## Features

- **NestJS Framework**: Modern, scalable Node.js framework
- **Fastify**: High-performance HTTP server
- **Drizzle ORM**: Type-safe SQL ORM
- **PostgreSQL**: Database
- **JWT Authentication**: Secure token-based authentication
- **WebSocket Support**: Real-time updates via Socket.IO
- **File Uploads**: Support for file uploads via Vercel Blob

## Project Structure

```
src/
├── auth/              # Authentication module
├── birthdays/         # Birthdays management
├── common/            # Shared utilities, guards, decorators
├── database/          # Database configuration and schema
├── families/          # Family/workspace management
├── folders/           # Folder organization
├── lists/             # Lists and list items
├── notes/             # Notes management
├── websocket/         # WebSocket gateway for real-time updates
├── app.module.ts      # Root application module
└── main.ts            # Application entry point
```

## Setup

1. **Install dependencies:**
   ```bash
   npm install
   ```

2. **Configure environment variables:**
   Create a `.env` file in the root directory:
   ```env
   DATABASE_URL=postgresql://user:password@localhost:5432/family_manager
   JWT_SECRET=your-secret-key-here
   BLOB_READ_WRITE_TOKEN=your-vercel-blob-token
   PORT=3000
   CORS_ORIGIN=http://localhost:5173,http://localhost:1420
   ```

3. **Run database migrations:**
   ```bash
   npm run db:push
   # or
   npm run db:migrate
   ```

4. **Start the development server:**
   ```bash
   npm run dev
   ```

## Available Scripts

- `npm run dev` - Start development server with hot reload
- `npm run build` - Build the application
- `npm run start` - Start production server
- `npm run start:prod` - Start production server (compiled)
- `npm run db:generate` - Generate database migrations
- `npm run db:migrate` - Run database migrations
- `npm run db:push` - Push schema changes to database
- `npm run db:studio` - Open Drizzle Studio

## API Endpoints

### Authentication
- `POST /auth/register` - Register a new user
- `POST /auth/login` - Login user
- `POST /auth/refresh` - Refresh JWT token
- `GET /auth/me` - Get current user

### Families
- `GET /families` - Get all families for user
- `GET /families/:id` - Get family details
- `POST /families` - Create new family
- `PUT /families/:id` - Update family
- `DELETE /families/:id` - Delete family
- `POST /families/:id/invite` - Invite member
- `POST /families/:id/join` - Join family
- `PUT /families/:id/members/:userId` - Update member role
- `DELETE /families/:id/members/:userId` - Remove member

### Folders
- `GET /folders` - Get folders (query: familyId, type)
- `GET /folders/:id` - Get folder details
- `POST /folders` - Create folder
- `PUT /folders/:id` - Update folder
- `DELETE /folders/:id` - Delete folder

### Lists
- `GET /lists` - Get lists (query: familyId, folderId, assignedTo)
- `GET /lists/:id` - Get list details
- `POST /lists` - Create list
- `PUT /lists/:id` - Update list
- `DELETE /lists/:id` - Delete list
- `POST /lists/:id/items` - Add list item
- `PUT /lists/:id/items/:itemId` - Update list item
- `DELETE /lists/:id/items/:itemId` - Delete list item

### Notes
- `GET /notes` - Get notes (query: familyId, folderId)
- `GET /notes/:id` - Get note details
- `POST /notes` - Create note
- `PUT /notes/:id` - Update note
- `DELETE /notes/:id` - Delete note
- `POST /notes/:id/upload` - Upload file to note

### Birthdays
- `GET /birthdays` - Get birthdays (query: familyId)
- `GET /birthdays/upcoming` - Get upcoming birthdays (query: familyId, limit)
- `GET /birthdays/:id` - Get birthday details
- `POST /birthdays` - Create birthday
- `PUT /birthdays/:id` - Update birthday
- `DELETE /birthdays/:id` - Delete birthday

## WebSocket Events

Connect to `/` (Socket.IO namespace) and authenticate:

```javascript
socket.emit('authenticate', { token: 'jwt-token', familyId: 'family-id' });
socket.emit('subscribe', { familyId: 'family-id' });
```

Events:
- `family_updated` - Family changes
- `folder_updated` - Folder changes
- `list_updated` - List changes
- `note_updated` - Note changes
- `birthday_updated` - Birthday changes

## Architecture

The application follows NestJS best practices:

- **Modular Architecture**: Each feature is a self-contained module
- **Dependency Injection**: Services are injected via constructors
- **DTOs**: Data Transfer Objects for request validation
- **Guards**: JWT authentication guard for protected routes
- **Services**: Business logic separated from controllers
- **Database Module**: Global database service for Drizzle ORM

## Code Quality

- TypeScript strict mode enabled
- Class-validator for DTO validation
- Consistent error handling
- Proper separation of concerns
- Well-documented code


