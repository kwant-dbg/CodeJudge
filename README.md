CodeJudge: A Microservices-Based Online Judge

CodeJudge is a high-performance, scalable online judge for competitive programming, built with a modern microservices architecture. It is designed to compile, run, and evaluate user-submitted code in a secure and resource-constrained environment.

This project is an excellent portfolio piece for backend engineering roles, demonstrating proficiency in multiple languages (Go, C++), database management (PostgreSQL, Redis), containerization (Docker), and complex system design.
Features

    Microservices Architecture: Each component is a separate service for scalability and maintainability.

    High-Performance Judge: The core judging logic is written in C++ for maximum performance and control over system resources.

    Secure Sandboxing: Uses setrlimit to enforce strict time and memory limits on user code.

    Asynchronous Job Queues: Leverages Redis for queuing submissions, ensuring the system can handle bursts of traffic.

    Plagiarism Detection: Includes a background service that uses a winnowing/shingling algorithm to detect similarities between submissions.

    Centralized API Gateway: A single entry point for all frontend requests, routing traffic to the appropriate internal service.

    Containerized: Fully containerized with Docker and Docker Compose for easy setup and deployment.

Tech Stack

    Backend Services: Go

    Judging Core: C++

    Database: PostgreSQL

    Message Queue: Redis

    Containerization: Docker, Docker Compose

Architecture Diagram
How to Run

    Prerequisites: Make sure you have Docker and Docker Compose installed on your machine.

    Clone the repository: git clone <your-repo-url>

    Build and run:

    docker-compose up --build

    The API Gateway will be available at http://localhost:8080.