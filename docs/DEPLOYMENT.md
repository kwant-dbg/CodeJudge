# CodeJudge Monolith Service - Docker Deployment

This guide covers deploying the CodeJudge Monolith service using Docker Compose.

## Quick Start

### Start the Monolith Service

```bash
docker-compose -f docker-compose.monolith.yml up -d --build
```

### Stop the Monolith Service

```bash
docker-compose -f docker-compose.monolith.yml down
```

### Stop and Remove Volumes (Clean Start)

```bash
docker-compose -f docker-compose.monolith.yml down -v
```

## Services

The deployment includes:

- **monolith** - The CodeJudge monolith service (Port 8080)
- **db** - PostgreSQL 14 database (Port 5432)
- **redis** - Redis 7 cache (Port 6379)

## Configuration

### Environment Variables

You can customize the deployment by modifying the environment variables in `docker-compose.monolith.yml`:

- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `JWT_SECRET` - Secret key for JWT token generation (change in production!)
- `PORT` - Service port (default: 8080)
- `GIN_MODE` - Gin framework mode (release/debug)

### Production Deployment

For production, make sure to:

1. Change the `JWT_SECRET` to a strong, random value
2. Update database credentials (POSTGRES_USER, POSTGRES_PASSWORD)
3. Consider using Docker secrets for sensitive data
4. Set up proper SSL/TLS termination (e.g., using a reverse proxy)
5. Configure backup strategies for PostgreSQL volumes

## Monitoring

### Check Service Status

```bash
docker-compose -f docker-compose.monolith.yml ps
```

### View Logs

```bash
# All services
docker-compose -f docker-compose.monolith.yml logs -f

# Monolith service only
docker-compose -f docker-compose.monolith.yml logs -f monolith

# Last 100 lines
docker-compose -f docker-compose.monolith.yml logs --tail=100 monolith
```

### Health Check

```bash
curl http://localhost:8080/health
```

### Ready Check

```bash
curl http://localhost:8080/ready
```

## Troubleshooting

### Container Won't Start

Check logs for detailed error messages:
```bash
docker-compose -f docker-compose.monolith.yml logs monolith
```

### Database Connection Issues

Ensure the database is healthy:
```bash
docker-compose -f docker-compose.monolith.yml ps db
docker-compose -f docker-compose.monolith.yml logs db
```

### Reset Everything

To completely reset the deployment:
```bash
docker-compose -f docker-compose.monolith.yml down -v
docker-compose -f docker-compose.monolith.yml up -d --build
```

## Accessing the Database

To connect to the PostgreSQL database directly:

```bash
docker-compose -f docker-compose.monolith.yml exec db psql -U user -d codejudgedb
```

## Accessing Redis

To connect to Redis:

```bash
docker-compose -f docker-compose.monolith.yml exec redis redis-cli
```

## API Endpoints

The monolith service exposes the following endpoints:

- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check endpoint
- `POST /auth/register` - User registration
- `POST /auth/login` - User login
- `GET /problems` - List problems
- `POST /problems` - Create problem (authenticated)
- `POST /submissions` - Submit solution
- `GET /submissions/:id` - Get submission status
- `POST /plagiarism/check` - Check plagiarism

## Volumes

The deployment creates persistent volumes:

- `postgres_data` - PostgreSQL data
- `redis_data` - Redis data

These volumes persist data across container restarts.

## Network

All services run in a dedicated Docker network (`codejudge_default`) and can communicate using service names:

- `db` - PostgreSQL database
- `redis` - Redis cache
- `monolith` - Monolith service

## Scaling

To scale the monolith service (requires load balancer):

```bash
docker-compose -f docker-compose.monolith.yml up -d --scale monolith=3
```

Note: You'll need to configure port mapping and load balancing for multiple instances.

## Updates

To update to the latest version:

```bash
git pull
docker-compose -f docker-compose.monolith.yml up -d --build
```
