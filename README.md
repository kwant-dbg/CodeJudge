# CodeJudge

**A high-performance backend for an online judge, built with a Go and C++ microservices architecture. It is designed to securely compile, run, and evaluate competitive programming submissions.**

## System Architecture

The system is built on a set of independent services that communicate asynchronously via a Redis message queue. All external traffic is routed through a single API Gateway.

### Submission Workflow:

`Client POST` -> `API Gateway` -> `Submissions Service` -> `[Redis Queues]` -> `Judge Service (C++) & Plagiarism Service (Go)` -> `PostgreSQL`

---

## Core Features

*   **Microservice Design:** A decoupled architecture promotes scalability and resilience. Each component is an independent service.
*   **High-Performance C++ Judge:** The core evaluation engine uses C++ for low-level process control, creating a secure sandbox with `fork()`, `exec()`, and `setrlimit`.
*   **Asynchronous Job Processing:** Redis queues allow the system to handle submission bursts gracefully, processing jobs in the background.
*   **Algorithmic Plagiarism Detection:** A background worker uses a fingerprinting pipeline (shingling/winnowing) to find logical similarities in code.
*   **Containerized Environment:** The entire application is containerized with Docker for consistent, one-command setup.

---

## Technology Stack

| Component | Technology |
| :--- | :--- |
| **Backend Services** | Go (net/http, go-redis, pq) |
| **Core Judge Engine**| C++17 |
| **Database** | PostgreSQL |
| **Message Queue** | Redis |
| **Containerization** | Docker, Docker Compose |

---

## Getting Started

### Prerequisites

*   Docker
*   Docker Compose

### Installation & Running

1.  **Clone the repository:**
    ```bash
    git clone <your-github-repository-url>
    cd codejudge
    ```

2.  **Build and run the services:**
    ```bash
    docker-compose up --build
    ```

The API Gateway will be available at `http://localhost:8080`.

---

## API Endpoints

All requests are routed through the API Gateway.

| Method | Path | Description |
| :--- | :--- | :--- |
| **GET** | `/api/problems/` | Retrieves a list of all programming problems. |
| **POST** | `/api/problems/` | Adds a new programming problem to the database. |
| **POST** | `/api/submissions`| Submits source code for a specific problem. |
| **GET** | `/api/plagiarism/reports` | Retrieves a dashboard of suspicious submissions. |

### Example Submission Request for `POST /api/submissions`:

```json
{
  "problem_id": 1,
  "source_code": "#include <iostream> int main() { int a,b; std::cin >> a >> b; std::cout << a+b; return 0; }"
}
```

---

## Service Breakdown

*   **`api-gateway`**: The single entry point that reverse-proxies requests to the appropriate internal microservice.
*   **`problems-service-go`**: Manages CRUD operations for programming problems.
*   **`submissions-service-go`**: Handles incoming code submissions, persists them, and pushes jobs to the Redis queues.
*   **`judge-service`**: The C++ core. It listens for jobs, compiles code, and executes it in a sandboxed environment to determine the verdict.
*   **`plagiarism-service-go`**: A background service that listens for jobs and performs similarity analysis between code submissions.
