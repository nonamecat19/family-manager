# Docker Setup Guide

This project uses Docker Compose to orchestrate the PostgreSQL database, NestJS backend, and Nginx reverse proxy.

## Architecture

- **PostgreSQL**: Database service running on port 5432 (internal)
- **Backend**: NestJS API service running on port 3000 (internal only)
- **Nginx**: Reverse proxy running on port 99 (external), routing requests to the backend

## Quick Start

1. **Create environment file** (optional, defaults are provided):
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Build and start all services**:
   ```bash
   docker-compose up -d
   ```

3. **View logs**:
   ```bash
   docker-compose logs -f
   ```

4. **Stop all services**:
   ```bash
   docker-compose down
   ```

## Access Points

- **Backend API**: `http://SERVE_IP:99/api/*`
  - Example: `http://localhost:99/api/auth/login`
  - Example: `http://localhost:99/api/families`
  - Swagger Docs: `http://localhost:99/api/docs`
  
- **Health Check**: `http://SERVE_IP:99/health`

- **PostgreSQL**: Direct access on port 5432 (if exposed)

## Environment Variables

Create a `.env` file in the project root with the following variables:

```env
# Server Configuration
SERVE_IP=localhost
NGINX_PORT=99

# Database Configuration
DATABASE_USER=family_manager
POSTGRES_PASSWORD=family_manager_password
POSTGRES_DB=family_manager
POSTGRES_PORT=5432

# Backend Configuration
JWT_SECRET=your-super-secret-jwt-key-change-in-production
CORS_ORIGIN=http://localhost:99,http://localhost:5173,http://localhost:1420
```

## Service Details

### PostgreSQL

- **Image**: `postgres:16-alpine`
- **Port**: 5432 (internal), configurable via `POSTGRES_PORT`
- **Data persistence**: Stored in `postgres_data` Docker volume
- **Health check**: Automatically checks if database is ready

### Backend

- **Build context**: `./backend`
- **Port**: 3000 (internal only, not exposed externally)
- **Health check**: Available at `/health`
- **Logs**: Available via `docker-compose logs backend`

### Nginx

- **Image**: `nginx:alpine`
- **Port**: 99 (external), configurable via `NGINX_PORT`
- **Configuration**: `./nginx/nginx.conf`
- **Routes**:
  - `/api/*` → Backend service
  - `/health` → Backend health check

## Development

### Rebuild after code changes

```bash
# Rebuild and restart backend
docker-compose up -d --build backend

# Or rebuild all services
docker-compose up -d --build
```

### Run database migrations

```bash
# Access backend container
docker-compose exec backend sh

# Inside container, run migrations
npm run db:migrate
```

### View service logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f postgres
docker-compose logs -f nginx
```

### Clean up

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (WARNING: deletes database data)
docker-compose down -v

# Remove images
docker-compose down --rmi all
```

## Troubleshooting

### Backend not starting

1. Check logs: `docker-compose logs backend`
2. Verify database is healthy: `docker-compose ps`
3. Check environment variables are set correctly

### Database connection errors

1. Ensure PostgreSQL service is healthy: `docker-compose ps postgres`
2. Verify `DATABASE_URL` is correct
3. Check network connectivity: `docker-compose exec backend ping postgres`

### Nginx routing issues

1. Check nginx logs: `docker-compose logs nginx`
2. Verify backend is healthy: `docker-compose ps backend`
3. Test backend directly: `docker-compose exec backend wget -O- http://localhost:3000/health`

### Port conflicts

If port 99 or 5432 is already in use, update `.env` file:
```env
NGINX_PORT=9999
POSTGRES_PORT=5433
```

## Production Considerations

1. **Security**:
   - Change all default passwords
   - Use strong JWT secret
   - Configure proper CORS origins
   - Use HTTPS (add SSL certificates to nginx)

2. **Performance**:
   - Adjust nginx worker connections
   - Configure database connection pooling
   - Enable nginx caching if needed

3. **Monitoring**:
   - Add logging aggregation
   - Set up health monitoring
   - Configure alerts

4. **Backups**:
   - Regularly backup PostgreSQL volume
   - Store backups securely


