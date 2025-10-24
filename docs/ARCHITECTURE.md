# CodeJudge Architecture Documentation

This document provides comprehensive architecture diagrams for the CodeJudge monolithic system.

## System Architecture Overview

```mermaid
graph TB
    subgraph "Client Layer"
        Browser[Web Browser]
        Mobile[Mobile Browser]
    end

    subgraph "Frontend - Static Files"
        HTML[HTML Pages<br/>index.html, contests.html<br/>problem.html, etc.]
        CSS[Styling<br/>Embedded CSS]
        JS[JavaScript<br/>API Calls, Dynamic UI]
    end

    subgraph "Monolith Service - Go"
        Router[Chi Router<br/>HTTP Multiplexer]
        
        subgraph "Middleware Layer"
            CORS[CORS Handler]
            Auth[JWT Auth]
            Logger[Request Logger]
            Recovery[Panic Recovery]
        end
        
        subgraph "Handlers"
            AuthH[Auth Handler<br/>Login/Register/Validate]
            ProblemsH[Problems Handler<br/>CRUD Operations]
            SubmissionsH[Submissions Handler<br/>Code Submission]
            ContestsH[Contests Handler<br/>Contest Management]
            PlagiarismH[Plagiarism Handler<br/>Code Similarity]
            AdminH[Admin Handler<br/>Dashboard]
        end
        
        subgraph "Common Libraries"
            DBUtil[DB Utilities<br/>Connection Pool]
            HTTPx[HTTP Helpers<br/>JSON Responses]
            RedisUtil[Redis Utilities]
            HealthCheck[Health Checks]
        end
    end

    subgraph "Judge Service - C++"
        JudgeWorker[Judge Worker<br/>modern_main.cpp]
        Sandbox[Sandboxing<br/>sandbox.cpp/h]
        Compiler[Compiler<br/>g++, python, javac]
    end

    subgraph "Data Layer"
        PostgreSQL[(PostgreSQL<br/>Problems, Users<br/>Submissions, Contests)]
        Redis[(Redis<br/>Job Queues<br/>Caching)]
        FileSystem[File System<br/>Submission Files]
    end

    Browser --> HTML
    Mobile --> HTML
    HTML --> Router
    JS --> Router
    
    Router --> CORS
    CORS --> Auth
    Auth --> Logger
    Logger --> Recovery
    Recovery --> AuthH
    Recovery --> ProblemsH
    Recovery --> SubmissionsH
    Recovery --> ContestsH
    Recovery --> PlagiarismH
    Recovery --> AdminH
    
    AuthH --> DBUtil
    ProblemsH --> DBUtil
    SubmissionsH --> DBUtil
    SubmissionsH --> RedisUtil
    ContestsH --> DBUtil
    PlagiarismH --> DBUtil
    PlagiarismH --> RedisUtil
    
    DBUtil --> PostgreSQL
    RedisUtil --> Redis
    SubmissionsH --> FileSystem
    
    RedisUtil --> JudgeWorker
    JudgeWorker --> Sandbox
    Sandbox --> Compiler
    JudgeWorker --> PostgreSQL
    JudgeWorker --> FileSystem

    style Browser fill:#667eea
    style Mobile fill:#667eea
    style Router fill:#10b981
    style PostgreSQL fill:#ef4444
    style Redis fill:#f59e0b
    style JudgeWorker fill:#8b5cf6
```

---

## Request Flow - User Submission

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Monolith
    participant PostgreSQL
    participant Redis
    participant Judge
    participant FileSystem

    User->>Browser: Write code & submit
    Browser->>Monolith: POST /api/submissions<br/>{code, problem_id, contest_id}
    
    Monolith->>Monolith: Validate JWT token
    Monolith->>Monolith: Extract user_id from token
    
    alt Contest Submission
        Monolith->>PostgreSQL: Get contest & problem points
        Monolith->>Monolith: Calculate points
    end
    
    Monolith->>PostgreSQL: INSERT submission<br/>(problem_id, user_id, code, contest_id, points)
    PostgreSQL-->>Monolith: Return submission_id
    
    Monolith->>FileSystem: Write code to file<br/>/app/submissions/{id}.cpp
    
    Monolith->>Redis: LPUSH submission_queue, id
    Monolith->>Redis: LPUSH plagiarism_queue, id
    
    Monolith-->>Browser: 201 Created<br/>{id, status: "Pending"}
    Browser-->>User: Show submission received
    
    loop Poll every 2s
        Browser->>Monolith: GET /api/submissions/{id}
        Monolith->>PostgreSQL: SELECT verdict FROM submissions
        Monolith-->>Browser: {verdict, status}
    end
    
    Note over Judge: Background Process
    Judge->>Redis: RPOP submission_queue
    Judge->>PostgreSQL: SELECT code, test_cases
    Judge->>FileSystem: Read code file
    Judge->>Judge: Compile & Execute in sandbox
    Judge->>PostgreSQL: UPDATE verdict
    
    Browser-->>User: Show verdict (Accepted/Wrong Answer/etc)
```

---

## Contest System Flow

```mermaid
flowchart TD
    Start([User Opens Contests Page]) --> LoadContests[GET /api/contests]
    LoadContests --> DisplayList[Display Contest List<br/>with Status Filters]
    
    DisplayList --> SelectContest{User Clicks<br/>Contest}
    SelectContest --> LoadDetails[GET /api/contests/:id]
    LoadDetails --> LoadProblems[GET /api/contests/:id/problems]
    LoadDetails --> LoadLeaderboard[GET /api/contests/:id/leaderboard]
    
    LoadProblems --> ShowContest[Display Contest Page<br/>Problems + Leaderboard]
    LoadLeaderboard --> ShowContest
    
    ShowContest --> CheckAuth{User<br/>Logged In?}
    CheckAuth -->|No| ShowLogin[Prompt Login]
    CheckAuth -->|Yes| RegisterBtn[Show Register Button]
    
    RegisterBtn --> Register{Register?}
    Register -->|Yes| PostRegister[POST /api/contests/:id/register]
    PostRegister --> Registered[User Registered]
    
    ShowContest --> SelectProblem{Click<br/>Problem}
    SelectProblem --> OpenProblem[Open Problem Page<br/>with contest_id parameter]
    
    OpenProblem --> ShowBanner[Show Contest Banner]
    ShowBanner --> WriteCode[User Writes Solution]
    WriteCode --> SubmitCode[Submit with contest_id]
    
    SubmitCode --> ProcessSubmission[POST /api/submissions<br/>contest_id included]
    ProcessSubmission --> CalcPoints[Calculate Points<br/>from contest_problems]
    CalcPoints --> SaveSubmission[(Save to DB<br/>with points)]
    
    SaveSubmission --> UpdateLeaderboard[Leaderboard Auto-Updates]
    UpdateLeaderboard --> ShowRank[Display User Rank]
    
    ShowContest --> AutoRefresh{Contest<br/>Active?}
    AutoRefresh -->|Yes| RefreshLoop[Refresh Leaderboard<br/>Every 30s]
    RefreshLoop --> CheckFreeze{Leaderboard<br/>Frozen?}
    CheckFreeze -->|Yes| ShowFrozen[Show Frozen Rankings]
    CheckFreeze -->|No| ShowLive[Show Live Rankings]
    ShowFrozen --> AutoRefresh
    ShowLive --> AutoRefresh
    AutoRefresh -->|No| End([Contest Ended])

    style Start fill:#667eea
    style End fill:#ef4444
    style ProcessSubmission fill:#10b981
    style UpdateLeaderboard fill:#f59e0b
```

---

## Database Schema

```mermaid
erDiagram
    USERS ||--o{ SUBMISSIONS : creates
    USERS ||--o{ CONTESTS : creates
    USERS ||--o{ CONTEST_PARTICIPANTS : registers
    
    PROBLEMS ||--o{ SUBMISSIONS : "solved by"
    PROBLEMS ||--o{ TEST_CASES : has
    PROBLEMS ||--o{ CONTEST_PROBLEMS : "included in"
    
    CONTESTS ||--o{ CONTEST_PROBLEMS : contains
    CONTESTS ||--o{ CONTEST_PARTICIPANTS : has
    CONTESTS ||--o{ SUBMISSIONS : receives
    
    SUBMISSIONS ||--o{ PLAGIARISM_REPORTS : "compared in"

    USERS {
        int id PK
        string username UK
        string email UK
        string password_hash
        string role
        timestamp created_at
    }

    PROBLEMS {
        int id PK
        string title
        text description
        string difficulty
        text input_format
        text output_format
    }

    TEST_CASES {
        int id PK
        int problem_id FK
        text input
        text output
        boolean sample
    }

    CONTESTS {
        int id PK
        string title
        text description
        timestamp start_time
        timestamp end_time
        int created_by FK
        int freeze_time
        timestamp created_at
    }

    CONTEST_PROBLEMS {
        int id PK
        int contest_id FK
        int problem_id FK
        int points
        int order_num
    }

    CONTEST_PARTICIPANTS {
        int contest_id FK
        int user_id FK
        timestamp joined_at
    }

    SUBMISSIONS {
        int id PK
        int problem_id FK
        int user_id FK
        int contest_id FK
        text source_code
        string language
        string verdict
        int points
        timestamp created_at
    }

    PLAGIARISM_REPORTS {
        int id PK
        int submission_a FK
        int submission_b FK
        float similarity
        timestamp created_at
    }
```

---

## Authentication & Authorization Flow

```mermaid
stateDiagram-v2
    [*] --> Unauthenticated
    
    Unauthenticated --> Registration : User registers
    Registration --> UserCreated : Hash password<br/>Store in DB
    
    Unauthenticated --> Login : User logs in
    Login --> ValidateCredentials : Check username<br/>Compare password hash
    
    ValidateCredentials --> LoginFailed : Invalid credentials
    ValidateCredentials --> GenerateJWT : Valid credentials
    
    LoginFailed --> Unauthenticated
    
    GenerateJWT --> Authenticated : Return JWT token<br/>Store in localStorage
    UserCreated --> Authenticated : Auto-login with JWT
    
    Authenticated --> MakeRequest : API call with token
    MakeRequest --> ValidateToken : Extract Bearer token<br/>Verify signature
    
    ValidateToken --> TokenInvalid : Expired/Invalid
    ValidateToken --> TokenValid : Valid token
    
    TokenInvalid --> Unauthenticated : Return 401
    
    TokenValid --> CheckRole : Extract user_id<br/>and role from claims
    
    CheckRole --> RegularUser : role = "user"
    CheckRole --> AdminUser : role = "admin"
    
    RegularUser --> AllowedAction : Can submit code<br/>View problems<br/>Register for contests
    AdminUser --> AllowedAction : All user actions +<br/>Create problems<br/>Create contests<br/>View plagiarism
    
    AllowedAction --> ProcessRequest : Execute handler
    ProcessRequest --> ReturnResponse : Return data
    
    ReturnResponse --> Authenticated
    
    Authenticated --> Logout : User logs out
    Logout --> [*] : Clear token
```

---

## Component Architecture

```mermaid
graph LR
    subgraph "Frontend Components"
        IndexPage[index.html<br/>Homepage & Problems]
        ProblemPage[problem.html<br/>Problem Details & IDE]
        ContestsPage[contests.html<br/>Contest Listing]
        ContestDetail[contest-detail.html<br/>Contest + Leaderboard]
        CreateContest[create-contest.html<br/>Admin: Create Contest]
        AdminPage[admin.html<br/>Admin Dashboard]
        PlagiarismPage[plagiarism.html<br/>Plagiarism Reports]
    end

    subgraph "API Handlers"
        AuthAPI[Auth API<br/>/api/auth/*]
        ProblemsAPI[Problems API<br/>/api/problems/*]
        SubmissionsAPI[Submissions API<br/>/api/submissions/*]
        ContestsAPI[Contests API<br/>/api/contests/*]
        PlagiarismAPI[Plagiarism API<br/>/api/plagiarism/*]
    end

    subgraph "Business Logic"
        UserMgmt[User Management<br/>Register, Login, JWT]
        ProblemMgmt[Problem Management<br/>CRUD, TestCases]
        JudgeMgmt[Judge Management<br/>Queue, Status]
        ContestMgmt[Contest Management<br/>Time, Scoring, Leaderboard]
        PlagiarismDetect[Plagiarism Detection<br/>MinHash LSH]
    end

    subgraph "Data Access"
        DBConn[Database Connection<br/>Connection Pool]
        RedisConn[Redis Connection<br/>Queue Management]
        FileAccess[File System<br/>Code Storage]
    end

    IndexPage --> ProblemsAPI
    ProblemPage --> ProblemsAPI
    ProblemPage --> SubmissionsAPI
    ContestsPage --> ContestsAPI
    ContestDetail --> ContestsAPI
    CreateContest --> ContestsAPI
    AdminPage --> AuthAPI
    PlagiarismPage --> PlagiarismAPI

    AuthAPI --> UserMgmt
    ProblemsAPI --> ProblemMgmt
    SubmissionsAPI --> JudgeMgmt
    ContestsAPI --> ContestMgmt
    PlagiarismAPI --> PlagiarismDetect

    UserMgmt --> DBConn
    ProblemMgmt --> DBConn
    JudgeMgmt --> DBConn
    JudgeMgmt --> RedisConn
    JudgeMgmt --> FileAccess
    ContestMgmt --> DBConn
    PlagiarismDetect --> DBConn
    PlagiarismDetect --> RedisConn

    style IndexPage fill:#667eea,color:#fff
    style ProblemPage fill:#667eea,color:#fff
    style ContestsPage fill:#667eea,color:#fff
    style ContestDetail fill:#667eea,color:#fff
    style DBConn fill:#ef4444,color:#fff
    style RedisConn fill:#f59e0b,color:#fff
```

## Deployment Architecture

```mermaid
graph LR
    Internet[Internet] --> Monolith[Monolith Container<br/>Go + Static Files<br/>Port 8080]
    
    Monolith --> DB[PostgreSQL 14<br/>Port 5432]
    Monolith --> Redis[Redis 7<br/>Port 6379]
    
    Judge[Judge Container<br/>C++ Worker] --> DB
    Judge --> Redis

    style Monolith fill:#10b981,color:#fff
    style Judge fill:#8b5cf6,color:#fff
    style DB fill:#ef4444,color:#fff
    style Redis fill:#f59e0b,color:#fff
```

---

## Leaderboard Calculation Logic

```mermaid
flowchart TD
    Start([Leaderboard Request]) --> GetContest[Fetch Contest Details<br/>start_time, end_time, freeze_time]
    
    GetContest --> CheckTime{Is Contest<br/>Active?}
    CheckTime -->|No| FinalStandings[Show Final Standings]
    CheckTime -->|Yes| CheckFreeze{Is Leaderboard<br/>Frozen?}
    
    CheckFreeze -->|No| QueryAll[Query All Submissions<br/>up to now]
    CheckFreeze -->|Yes| QueryFrozen[Query Submissions<br/>up to freeze_time]
    
    QueryAll --> ProcessScores
    QueryFrozen --> ProcessScores
    
    ProcessScores[Calculate Per User:<br/>- Total Score = SUM points for Accepted<br/>- Problems Solved = COUNT DISTINCT problems<br/>- Penalty = Minutes from start to submission]
    
    ProcessScores --> RankUsers[Rank Users by:<br/>1. Total Score DESC<br/>2. Problems Solved DESC<br/>3. Last Submission ASC]
    
    RankUsers --> FormatResponse[Format Leaderboard:<br/>- Rank<br/>- Username<br/>- Score<br/>- Problems Solved<br/>- Penalty]
    
    FormatResponse --> AddMetadata[Add Metadata:<br/>- is_frozen flag<br/>- contest_id]
    
    AddMetadata --> ReturnJSON[Return JSON Response]
    ReturnJSON --> End([Client Displays Leaderboard])
    
    FinalStandings --> ProcessScores

    style Start fill:#667eea,color:#fff
    style End fill:#10b981,color:#fff
    style CheckFreeze fill:#f59e0b
    style RankUsers fill:#8b5cf6,color:#fff
```

---

## Security Layers

```mermaid
graph TD
    Request[Incoming HTTP Request]
    
    Request --> HTTPS{HTTPS?}
    HTTPS -->|No| Reject1[Reject in Production]
    HTTPS -->|Yes| CORS_Check[CORS Validation]
    
    CORS_Check --> ValidOrigin{Valid Origin?}
    ValidOrigin -->|No| Reject2[403 Forbidden]
    ValidOrigin -->|Yes| AuthCheck[Check Authorization Header]
    
    AuthCheck --> HasToken{Has Bearer<br/>Token?}
    HasToken -->|No Public Route| ProcessPublic[Allow Public Access]
    HasToken -->|No Protected Route| Reject3[401 Unauthorized]
    HasToken -->|Yes| ValidateJWT[Validate JWT Signature]
    
    ValidateJWT --> JWTValid{JWT Valid?}
    JWTValid -->|No| Reject4[401 Invalid Token]
    JWTValid -->|Yes| ExtractClaims[Extract Claims<br/>user_id, role, exp]
    
    ExtractClaims --> CheckExpiry{Token<br/>Expired?}
    CheckExpiry -->|Yes| Reject5[401 Token Expired]
    CheckExpiry -->|No| CheckRole{Requires<br/>Admin?}
    
    CheckRole -->|Yes| IsAdmin{role ==<br/>"admin"?}
    CheckRole -->|No| AddContext[Add user_id to Context]
    
    IsAdmin -->|No| Reject6[403 Forbidden]
    IsAdmin -->|Yes| AddContext
    
    AddContext --> InputValidation[Validate Input<br/>SQL Injection Prevention<br/>XSS Prevention]
    
    InputValidation --> RateLimiting[Rate Limiting Check]
    RateLimiting --> ProcessRequest[Process Request]
    
    ProcessRequest --> Sandbox{Code<br/>Execution?}
    Sandbox -->|Yes| SecureSandbox[Run in Secure Sandbox<br/>Limited Memory<br/>Limited Time<br/>No Network]
    Sandbox -->|No| DBQuery[Execute Database Query]
    
    SecureSandbox --> Response
    DBQuery --> Response[Return Response]
    
    ProcessPublic --> Response
    
    Response --> End([Send to Client])

    style Request fill:#667eea,color:#fff
    style Response fill:#10b981,color:#fff
    style Reject1 fill:#ef4444,color:#fff
    style Reject2 fill:#ef4444,color:#fff
    style Reject3 fill:#ef4444,color:#fff
    style Reject4 fill:#ef4444,color:#fff
    style Reject5 fill:#ef4444,color:#fff
    style Reject6 fill:#ef4444,color:#fff
    style SecureSandbox fill:#8b5cf6,color:#fff
```

---

## Technology Stack

```mermaid
mindmap
  root((CodeJudge<br/>Monolith))
    Backend
      Go 1.23
        Chi Router
        JWT Auth
        Bcrypt
      C++ Judge
        Modern C++17
        Sandboxing
        Compiler Support
    Frontend
      HTML5
      Vanilla JavaScript
      CSS3
        Responsive Design
        Dark Mode
      KaTeX
        LaTeX Rendering
    Database
      PostgreSQL 14
        ACID Transactions
        Indexes
        Foreign Keys
      Redis 7
        Job Queues
        Caching
    DevOps
      Docker
      Docker Compose
      Health Checks
      Volumes
    Libraries
      go-chi/chi
      golang-jwt
      go-redis
      libpq
      hiredis
```

---

## Scaling Considerations

```mermaid
graph TB
    subgraph "Current Monolith"
        M[Single Monolith Instance<br/>All services in one]
    end
    
    subgraph "Future: Horizontal Scaling"
        LB[Load Balancer]
        M1[Monolith Instance 1]
        M2[Monolith Instance 2]
        M3[Monolith Instance 3]
        
        LB --> M1
        LB --> M2
        LB --> M3
    end
    
    subgraph "Future: Microservices (Optional)"
        Gateway[API Gateway]
        AuthMS[Auth Service]
        ProblemMS[Problems Service]
        SubmitMS[Submissions Service]
        ContestMS[Contests Service]
        JudgeMS[Judge Service Pool]
        
        Gateway --> AuthMS
        Gateway --> ProblemMS
        Gateway --> SubmitMS
        Gateway --> ContestMS
        SubmitMS --> JudgeMS
    end
    
    subgraph "Shared Services"
        DB[(PostgreSQL<br/>Read Replicas)]
        RedisCluster[(Redis Cluster)]
        S3[(Object Storage<br/>Submissions)]
    end
    
    M --> DB
    M --> RedisCluster
    
    M1 --> DB
    M2 --> DB
    M3 --> DB
    M1 --> RedisCluster
    M2 --> RedisCluster
    M3 --> RedisCluster
    
    AuthMS --> DB
    ProblemMS --> DB
    SubmitMS --> DB
    ContestMS --> DB
    JudgeMS --> DB
    JudgeMS --> RedisCluster
    SubmitMS --> S3

    style M fill:#10b981,color:#fff
    style M1 fill:#10b981,color:#fff
    style M2 fill:#10b981,color:#fff
    style M3 fill:#10b981,color:#fff
    style Gateway fill:#667eea,color:#fff
```

---

## Notes

- All diagrams are in Mermaid format for native GitHub rendering
- Copy any diagram directly into your README.md
- Diagrams automatically render on GitHub
- Can be viewed in VS Code with Mermaid extensions
- Export to PNG/SVG using Mermaid Live Editor: https://mermaid.live
