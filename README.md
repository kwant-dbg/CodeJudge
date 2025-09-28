CodeJudge: A High-Performance Microservices-Based Online Judge

CodeJudge is a scalable backend system for a competitive programming platform, engineered with a modern microservices architecture. It is designed to compile, run, and evaluate user-submitted code within a secure, resource-constrained sandbox environment. The system also includes a background pipeline for algorithmic plagiarism detection.

This project serves as a comprehensive demonstration of backend development principles, including system design, inter-service communication, containerization, and the integration of multiple programming languages (Go and C++) to leverage their respective strengths.
System Architecture

The system is composed of several independent services that communicate via HTTP and a Redis message queue. An API Gateway acts as the single entry point for all external requests, routing them to the appropriate internal service. This design ensures separation of concerns and allows for individual services to be scaled or updated independently.

The typical workflow is as follows:

    A user submits code via the API Gateway.

    The Submissions Service saves the submission to the PostgreSQL database.

    The service then pushes the submission ID onto two Redis queues: one for judging and one for plagiarism detection.

    The C++ Judge Service and Go-based Plagiarism Service concurrently pull jobs from their respective queues.

    The Judge Service compiles and executes the code against test cases, enforcing strict time and memory limits.

    The Plagiarism Service generates a fingerprint of the code and compares it to other submissions for the same problem.

    Results and reports are stored back in the database.

Key Features

    Microservice Design: The system is logically divided into services for handling problems, submissions, judging, and plagiarism, promoting scalability and resilience.

    High-Performance C++ Judge: The core evaluation engine is written in C++ for low-level control over process execution. It uses fork(), exec(), and setrlimit to create a secure sandbox for user code.

    Asynchronous Job Processing: Redis is used as a message broker to decouple services. This allows the system to handle submission bursts gracefully, processing jobs asynchronously.

    Algorithmic Plagiarism Detection: A background worker implements a fingerprinting pipeline using shingling and winnowing algorithms to identify logical similarities between code submissions.

    Containerized Environment: The entire application is containerized using Docker and orchestrated with Docker Compose, allowing for consistent, one-command setup and deployment.

Technology Stack

Component
	

Technology

Backend Services
	

Go (net/http, go-redis, pq)

Core Judge Engine
	

C++17

Database
	

PostgreSQL

Message Queue / Caching
	

Redis

Containerization
	

Docker, Docker Compose
Getting Started
Prerequisites

    Docker

    Docker Compose

Installation & Running

    Clone the repository to your local machine.

    git clone <your-github-repository-url>
    cd codejudge

    Build and start the services using Docker Compose.

    docker-compose up --build

    The services will be built and started. The API Gateway is now accessible at http://localhost:8080.

API Endpoints

All endpoints are accessed through the API Gateway.

Method
	

Endpoint
	

Description

GET
	

/api/problems/
	

Retrieves a list of all programming problems.

POST
	

/api/problems/
	

Adds a new programming problem to the database.

POST
	

/api/submissions
	

Submits source code for a specific problem.

GET
	

/api/plagiarism/reports
	

Retrieves a dashboard of submissions with high similarity scores.

Example Submission Request:
POST /api/submissions

{
  "problem_id": 1,
  "source_code": "#include <iostream> int main() { int a,b; std::cin >> a >> b; std::cout << a+b; return 0; }"
}

Service Breakdown

    api-gateway: The single entry point for all client requests. It reverse-proxies requests to the appropriate internal microservice.

    problems-service-go: Manages the CRUD operations for programming problems stored in the PostgreSQL database.

    submissions-service-go: Handles incoming code submissions, persists them, and places new jobs onto the Redis queues for judging and plagiarism analysis.

    judge-service: The C++ core of the system. It listens for jobs on the submission_queue, compiles the user's code, executes it in a sandboxed environment, and determines the verdict.

    plagiarism-service-go: A background service that listens for jobs on the plagiarism_queue and performs similarity analysis between code submissions.