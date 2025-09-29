# CodeJudge: A High-Performance Online Judging System

CodeJudge is a scalable, cloud-native backend designed to power competitive programming platforms, educational coding websites, and automated technical screening. It provides a secure, high-performance environment for compiling, executing, and evaluating user-submitted code against predefined test cases.

Built with a distributed microservices architecture using Go and C++, CodeJudge is engineered for reliability, scalability, and efficient resource management.

![System Architecture Diagram](https://github.com/user-attachments/assets/9ab15fcd-070d-46b2-84ae-07ee72f307a)

## Key Differentiators

*   **Security-First Sandboxing:** The core C++ judge engine leverages low-level Linux primitives (`fork`, `exec`, `setrlimit`) to create a secure, isolated sandbox for code execution, preventing malicious submissions from affecting the host system.
*   **Horizontally Scalable:** The decoupled, message-driven architecture allows any component (e.g., judge workers, plagiarism checkers) to be scaled independently to handle fluctuating loads.
*   **Advanced Plagiarism Detection:** Goes beyond simple text matching by using a sophisticated fingerprinting pipeline (shingling and winnowing) to detect underlying algorithmic similarities between submissions.
*   **Developer-Friendly & Extensible:** The entire system is containerized with Docker and managed via a single `docker-compose` file, allowing for a one-command setup. Its modular design makes it easy to extend or integrate with existing platforms.

## Core Features

*   **Microservice Architecture**: Decoupled Go and C++ services communicate asynchronously via a Redis message queue for enhanced resilience and scalability.
*   **High-Performance C++ Judge**: A lightweight, high-throughput C++ engine for low-overhead process control and sandboxing.
*   **Asynchronous Job Queues**: Utilizes Redis to manage submission workloads, ensuring smooth processing during peak traffic and enabling easy retry logic.
*   **Algorithmic Plagiarism Detection**: A background service dedicated to identifying logical similarities in submitted code.
*   **Containerized Deployment**: Fully containerized with Docker for consistent, cross-platform deployments.

## Technology Stack

| Component             | Technology                               | Purpose                                          |
| --------------------- | ---------------------------------------- | ------------------------------------------------ |
| **Backend Services**  | Go (net/http, go-redis, pq)              | API endpoints, business logic, and service orchestration |
| **Core Judge Engine** | C++17                                    | Secure code execution and sandboxing             |
| **Database**          | PostgreSQL                               | Data persistence for submissions, problems, and reports |
| **Message Queue**     | Redis                                    | Asynchronous job distribution between services   |
| **Containerization**  | Docker, Docker Compose                   | Environment consistency and simplified deployment |

## Getting Started

### Prerequisites

*   Docker Engine
*   Docker Compose

### Quickstart

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/kwant-dbg/CodeJudge.git
    cd CodeJudge
    ```

2.  **Launch the services:**
    ```bash
    docker-compose up --build -d
    ```

The API Gateway is now available at `http://localhost:8080`.

## API Endpoints

All endpoints are consolidated under the API Gateway.

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/problems/` | Retrieve a list of all programming problems. |
| `POST` | `/api/problems/` | Add a new programming problem. |
| `POST` | `/api/submissions` | Submit source code for evaluation against a problem. |
| `GET` | `/api/plagiarism/reports` | Retrieve a dashboard of submissions with high similarity scores. |

### Example Submission Request: `POST /api/submissions`

```json
{
  "problem_id": 1,
  "language": "cpp",
  "source_code": "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b << std::endl; return 0; }"
}
```
