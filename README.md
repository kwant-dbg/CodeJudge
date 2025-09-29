### CodeJudge: A High-Performance Online Judging System

CodeJudge is a scalable, cloud-native backend for competitive programming platforms and automated technical screening. It provides a secure, high-performance environment to compile, execute, and evaluate user-submitted code against predefined test cases. Its distributed microservices architecture, built with Go and C++, is designed for reliability and efficiency.

<img width="1024" height="1024" alt="29, 2025 - 12_10PM" src="https://github.com/user-attachments/assets/9ab15fcd-070d-46b2-84ae-07ee72f3b07a" />

### Core Features

*   **Secure Sandboxing:** The C++ judge engine uses low-level Linux primitives (`fork`, `exec`, `setrlimit`) to securely isolate code execution.
*   **Scalable Architecture:** A decoupled, message-driven design allows any component to be scaled independently to handle fluctuating loads.
*   **Advanced Plagiarism Detection:** A fingerprinting pipeline (shingling and winnowing) detects algorithmic similarities between submissions.
*   **Easy Deployment:** The entire system is containerized with Docker and managed via a single `docker-compose` file for a one-command setup.

### Technology Stack

| Component | Technology | Purpose |
| :--- | :--- | :--- |
| **Backend Services** | Go (net/http, go-redis, pq) | API, business logic, service orchestration |
| **Core Judge Engine**| C++17 | Secure code execution and sandboxing |
| **Database** | PostgreSQL | Data persistence |
| **Message Queue** | Redis | Asynchronous job distribution |
| **Containerization**| Docker, Docker Compose | Consistent and simple deployment |

### Quickstart

**Prerequisites:** Docker and Docker Compose.

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/kwant-dbg/CodeJudge.git
    cd CodeJudge
    ```

2.  **Launch the services:**
    ```bash
    docker-compose up --build -d
    ```
    The API Gateway will be available at `http://localhost:8080`.

### API Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/problems/` | Get all programming problems. |
| `POST`| `/api/problems/` | Add a new programming problem. |
| `POST`| `/api/submissions`| Submit code for evaluation. |
| `GET` | `/api/plagiarism/reports` | Get a report of submissions with high similarity. |

**Example Submission:** `POST /api/submissions`
```json
{
  "problem_id": 1,
  "language": "cpp",
  "source_code": "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b << std::endl; return 0; }"
}
```
