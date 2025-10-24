# CodeJudge

<p align="center">

![CodeJudge](https://img.shields.io/badge/CodeJudge-Online%20Judge-667eea?style=for-the-badge)
![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=for-the-badge&logo=go)
![C++](https://img.shields.io/badge/C++-17-00599C?style=for-the-badge&logo=cplusplus)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14-336791?style=for-the-badge&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker)

A production-ready online judge system featuring real-time contest management, automated plagiarism detection, and secure sandboxed code execution.

[Features](#features) • [Quick Start](#quick-start) • [Architecture](#architecture) • [Documentation](#documentation) • [API](#api-reference)

</p>

---

## Features

- **Problem Management** - Create and manage coding problems with LaTeX support
- **Multi-Language Support** - C++, Python, and Java compilation
- **Secure Sandbox** - Isolated execution with resource limits
- **Contests** - Time-based contests with live leaderboards and freeze periods
- **Plagiarism Detection** - MinHash LSH algorithm for code similarity
- **Modern UI** - Responsive design with dark mode and real-time updates
- **JWT Authentication** - Secure token-based authentication with role-based access
- **Admin Dashboard** - Centralized platform management

---

## Quick Start

### Prerequisites
- Docker 20.10+ and Docker Compose 2.0+
- Minimum 4GB RAM
- 10GB available disk space

### Installation

```bash
# Clone the repository
git clone https://github.com/kwant-dbg/CodeJudge.git
cd CodeJudge

# Start all services
docker-compose up -d --build

# Access the platform
# Navigate to http://localhost:8080 in your browser
```

The application will be available at `http://localhost:8080` once all containers are running.

### Initial Setup

1. Register an account through the registration interface
2. Browse the problem set and review available challenges
3. Submit solutions for automated evaluation
4. Participate in scheduled contests
5. (Administrator) Access administrative functions for contest and problem management

---

## Architecture

CodeJudge uses a **monolithic architecture** for simplicity and performance:

```mermaid
graph TB
    subgraph "Client Layer"
        Browser[Web Browser]
    end

    subgraph "Monolith Service - Go"
        Router[Chi Router]
        Auth[JWT Auth]
        Handlers[API Handlers<br/>Problems, Submissions<br/>Contests, Plagiarism]
    end

    subgraph "Judge Service - C++"
        Judge[Judge Worker]
        Sandbox[Secure Sandbox]
    end

    subgraph "Data Layer"
        PostgreSQL[(PostgreSQL<br/>Data Storage)]
        Redis[(Redis<br/>Job Queue)]
    end

    Browser --> Router
    Router --> Auth
    Auth --> Handlers
    Handlers --> PostgreSQL
    Handlers --> Redis
    Redis --> Judge
    Judge --> Sandbox
    Judge --> PostgreSQL

    style Browser fill:#667eea,color:#fff
    style Router fill:#10b981,color:#fff
    style PostgreSQL fill:#ef4444,color:#fff
    style Redis fill:#f59e0b,color:#fff
    style Judge fill:#8b5cf6,color:#fff
```

**📚 Detailed Architecture:** See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for comprehensive diagrams including:
- System architecture overview
- Request flow diagrams
- Database schema (ERD)
- Authentication flow
- Contest system logic
- Security layers
- Deployment architecture


---

## Project Structure

```
codejudge/
├── monolith/                 # Go Backend Service
│   ├── handlers/             # HTTP Request Handlers
│   │   ├── admin.go             # Administrative dashboard
│   │   ├── auth.go              # JWT authentication & user management
│   │   ├── contests.go          # Contest lifecycle & leaderboards
│   │   ├── plagiarism.go        # MinHash LSH similarity detection
│   │   ├── problems.go          # Problem CRUD with direct SQL queries
│   │   └── submissions.go       # Code submission & queue management
│   ├── static/               # Frontend HTML/CSS/JS
│   │   ├── admin.html           # Admin dashboard
│   │   ├── contest-detail.html  # Live contest leaderboard
│   │   ├── contests.html        # Contest listing page
│   │   ├── create-contest.html  # Admin contest creation
│   │   ├── create-problem.html  # Admin problem creation
│   │   ├── index.html           # Homepage & navigation
│   │   ├── plagiarism.html      # Plagiarism report interface
│   │   ├── problem.html         # Problem viewer with LaTeX & submission form
│   │   └── submission.html      # Submission status viewer
│   ├── main.go               # Application entrypoint & routing
│   ├── startup.sh            # Container startup script
│   └── Dockerfile.standalone # Docker build configuration
│
├── judge/                    # C++ Judge Service (Sandboxed Execution)
│   ├── modern_main.cpp          # Judge worker with Redis queue consumer
│   ├── sandbox.cpp              # Secure sandbox implementation
│   ├── sandbox.h                # Sandbox interface definitions
│   ├── CMakeLists.txt           # Build configuration
│   └── Dockerfile.modern        # Docker build with dependencies
│
├── common/                   # Shared Go Libraries
│   ├── auth/                    # JWT token generation & validation
│   ├── dbutil/                  # Database connection pooling (simplified)
│   │   ├── connection_manager.go
│   │   └── db.go
│   ├── env/                     # Environment variable helpers
│   ├── health/                  # Health check endpoints
│   ├── httpx/                   # HTTP utilities & middleware
│   └── redisutil/               # Redis queue management
│
├── docs/                     # Documentation
│   ├── ARCHITECTURE.md          # System architecture & diagrams
│   ├── CONTESTS_FEATURE.md      # Contest system documentation
│   └── DEPLOYMENT.md            # Deployment instructions
│
├── deploy/                   # Deployment Scripts
│   ├── deploy-to-azure.ps1      # Azure deployment
│   ├── local.ps1                # Local Docker management
│   └── seed-db.sql              # Database seeding with sample problems
│
└── docker-compose.yml        # Local development environment
```
## How It Works

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Monolith
    participant Redis
    participant Judge
    participant DB

    User->>Frontend: Submit Code
    Frontend->>Monolith: POST /api/submissions
    Monolith->>DB: Save Submission (PENDING)
    Monolith->>Redis: Push to submission_queue
    Monolith-->>Frontend: 202 Accepted
    
    Judge->>Redis: Pull Job
    Judge->>Judge: Compile & Execute<br/>(Sandboxed)
    Judge->>DB: Update Result (AC/WA/TLE/etc)
    
    Frontend->>Monolith: GET /api/submissions/:id
    Monolith->>DB: Fetch Status
    Monolith-->>Frontend: Return Verdict
    Frontend->>User: Display Result
```

**Submission Processing Flow:**
1. Client submits source code via REST API
2. Monolith service persists submission to PostgreSQL with PENDING status
3. Job enqueued to Redis for asynchronous processing
4. Judge worker dequeues job and executes in isolated sandbox
5. Test cases evaluated with enforced time and memory constraints
6. Verdict stored in database with detailed execution metrics
7. Client retrieves results via polling or webhook

---

## Development

### Local Development Environment

```bash
# Install Go dependencies
cd monolith
go mod download

# Run monolith service (requires PostgreSQL and Redis running)
go run main.go

# Compile judge service
cd ../judge
g++ -std=c++17 modern_main.cpp sandbox.cpp -o judge
```

### Database Setup

```bash
# Initialize database with sample data
cd deploy
./seed-db.sh

# Manual initialization
psql -U postgres -d codejudge -f seed-db.sql
```

### Docker Development Workflow

```bash
# Rebuild specific service
docker-compose build monolith
docker-compose up -d monolith

# Monitor service logs
docker-compose logs -f monolith judge

# Clean rebuild
docker-compose down -v
docker-compose up -d --build
```

---

## Testing

```bash
# Execute full test suite
cd monolith
go test ./...

# Run specific package tests
go test ./handlers -v

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Documentation

- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design, database schema, request flows
- **[Contest System](docs/CONTESTS_FEATURE.md)** - Contest management, leaderboard API, scoring logic
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Docker deployment, Azure setup, production config

---

## API Reference

### Authentication
- `POST /api/register` - Create new user
- `POST /api/login` - Get JWT token
- `POST /api/validate` - Validate JWT
- `GET /api/me` - Get current user info

### Problems
- `GET /api/problems` - List all problems
- `GET /api/problems/:id` - Get problem details
- `POST /api/problems` - Create problem (admin)
- `POST /api/problems/:id/testcases` - Add test cases (admin)

### Submissions
- `POST /api/submissions` - Submit solution
- `GET /api/submissions/:id` - Get submission status

### Contests
- `GET /api/contests` - List contests
- `POST /api/contests` - Create contest (admin)
- `GET /api/contests/:id` - Get contest details
- `POST /api/contests/:id/register` - Register for contest
- `GET /api/contests/:id/leaderboard` - Live leaderboard
- `POST /api/contests/:id/problems` - Add problem to contest (admin)

### Plagiarism
- `GET /api/plagiarism/reports` - Get plagiarism reports (admin)

**Example Submission:**
```json
{
  "problem_id": 1,
  "language": "cpp",
  "source_code": "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b; return 0; }"
}
```

**Full API documentation:** See [ARCHITECTURE.md](docs/ARCHITECTURE.md#api-endpoints)

---

## Roadmap

Planned features and enhancements:

- [ ] Virtual contest mode for practice
- [ ] User profile pages with detailed statistics
- [ ] Community discussion forums

---

## Author

**Harshit Sharma**
- GitHub: [@kwant-dbg](https://github.com/kwant-dbg)
