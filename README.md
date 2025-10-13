# CodeJudge (Mono Branch)

CodeJudge is a high-performance online judge system. The **mono branch** uses a monolithic Go application (with C++ judge) instead of a distributed microservices architecture.

---

## Mono Branch Highlights

- **Monolith-first:** All backend logic is in a single Go service (`monolith/`), simplifying deployment and development.
- **C++ Judge:** Secure sandboxed code execution using a C++ engine (`judge/`).
- **PostgreSQL & Redis:** Used for persistence and job queueing.
- **Dockerized:** One `docker-compose.yml` to run everything locally or in the cloud.

---

## Quick Start

**Prerequisites:** Docker, Docker Compose

```powershell
# Start all services (monolith, judge, db, redis)
docker-compose up -d --build

# Access the web UI at http://localhost:8080
```

---

## Project Structure (Mono Branch)

```
codejudge/
   monolith/          # Main Go application (all API logic)
   judge/             # C++ judge service (sandboxed execution)
   common/            # Shared Go libraries/utilities
   deploy/            # Deployment scripts
   docs/              # Documentation
   docker-compose.yml # Main compose file
```

---

## Deployment & Management

See `docs/DEPLOYMENT.md` for full details.

**Start:**
```powershell
docker-compose up -d --build
```

**Stop:**
```powershell
docker-compose down
```

**Logs:**
```powershell
docker-compose logs -f
```

**Database/Redis access:**
```powershell
docker-compose exec db psql -U user -d codejudgedb
docker-compose exec redis redis-cli
```

---

## API Endpoints (Monolith)

| Method | Path | Description |
|--------|------|-------------|
| GET    | /api/problems/           | List all problems |
| GET    | /api/problems/{id}       | Get problem by ID |
| POST   | /api/problems/           | Create a new problem (auth required) |
| POST   | /api/problems/{id}/testcases | Add testcases (auth required) |
| POST   | /api/auth/register       | Register a new user |
| POST   | /api/auth/login          | Login and get JWT |
| POST   | /api/auth/validate       | Validate JWT |
| GET    | /api/auth/me             | Get current user info (auth required) |
| POST   | /api/submissions         | Submit code for judging (auth required) |
| GET    | /api/submissions/{id}    | Get submission result (auth required) |
| GET    | /api/plagiarism/reports  | Get plagiarism reports (auth required) |

---

## Example Submission

```json
{
   "problem_id": 1,
   "language": "cpp",
   "source_code": "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b << std::endl; return 0; }"
}
```

---

## Development & Testing

- All backend logic is in `monolith/` (Go)
- Judge logic is in `judge/` (C++)
- Shared code/utilities in `common/`

---

## Documentation

- [Deployment Guide](docs/DEPLOYMENT.md)
- [Azure Quickstart](docs/AZURE_QUICKSTART.md)
- [Azure Deployment](docs/AZURE.md)
- [Sample Problems](docs/SAMPLE_PROBLEMS.md)
- [Sample Solutions](docs/SAMPLE_SOLUTIONS.md)
- [Sample Test Cases](docs/SAMPLE_TESTS_COMPLETE.md)
- [Codeforces-Style Update](docs/CODEFORCES_STYLE_UPDATE.md)

> Internal notes, refactoring logs, and LaTeX improvement docs are not included in the public repository.

---

## Maintainer

Harshit Sharma
