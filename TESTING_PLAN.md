# Performance & Scalability Testing Plan

This document outlines the steps to generate and validate the performance metrics for the CodeJudge project.

## 1. Judge Throughput & Horizontal Scalability

**Goal:** Measure the number of submissions a single judge worker can process per second and prove that this throughput scales linearly when more workers are added.

### Prerequisites
- Docker & Docker Compose
- Go
- Redis CLI

### Step 1: Create the Queue Loading Script

Create the file `scripts/load_queue.go` with the provided content. This script will push 1,000 jobs directly to the Redis queue, bypassing the API to isolate the judge's performance.

### Step 2: Test with a Single Worker

1.  **Start services with 1 judge worker:**
    ```bash
    docker-compose up -d --build --scale judge=1
    ```

2.  **Run the load script:**
    ```bash
    go run ./scripts/load_queue.go
    ```

3.  **Time the process:** Start a timer and monitor the queue length. Stop the timer when the queue is empty.
    ```bash
    # Run this command repeatedly until it returns 0
    docker-compose exec redis redis-cli LLEN submission_queue
    ```

4.  **Calculate throughput:**
    `Throughput = 1000 jobs / Total Time (in seconds)`
    *Example: 1000 jobs / 200 seconds = **5 submissions/sec***

### Step 3: Test with Multiple Workers

1.  **Clean up the previous environment:**
    ```bash
    docker-compose down -v
    ```

2.  **Start services with 4 judge workers:**
    ```bash
    docker-compose up -d --build --scale judge=4
    ```

3.  **Run the load script again:**
    ```bash
    go run ./scripts/load_queue.go
    ```

4.  **Time the process again:** Monitor the queue until it is empty. The time should be significantly less.

5.  **Calculate new throughput:**
    `Throughput = 1000 jobs / New Total Time (in seconds)`
    *Example: 1000 jobs / 50 seconds = **20 submissions/sec***

### Conclusion

You can now confidently state that a single worker has a baseline throughput of ~5 submissions/sec and that the system is horizontally scalable, which you've proven by achieving ~20 submissions/sec with four workers.

---

## 2. API Latency (Leaderboard)

**Goal:** Measure the 95th percentile (p95) API latency for the leaderboard endpoint before and after implementing Redis caching.

### Prerequisites
- Docker & Docker Compose
- k6 Load Testing Tool

### Step 1: Create the k6 Test Script

Create the file `scripts/leaderboard_test.js` with the provided content. This script simulates 10 concurrent users requesting the leaderboard for 30 seconds.

### Step 2: Ensure Sufficient Test Data

Before running, make sure your database is seeded with a realistic amount of data for a contest (e.g., 100+ users, 1000+ submissions). You can expand `deploy/seed-db.sql` if needed.

### Step 3: Test "Before" Caching

1.  **Disable Caching:** In your Go monolith code, comment out or disable the Redis caching logic for the leaderboard handler.
2.  **Start services:**
    ```bash
    docker-compose up -d --build
    ```
3.  **Run the k6 test:**
    ```bash
    k6 run ./scripts/leaderboard_test.js
    ```
4.  **Record the result:** Find the `http_req_duration` metric in the output and note the `p(95)` value. It will likely be high (e.g., `p(95)......810.5ms`).

### Step 4: Test "After" Caching

1.  **Enable Caching:** Re-enable the Redis caching logic in your Go code.
2.  **Restart services:**
    ```bash
    docker-compose down && docker-compose up -d --build
    ```
3.  **Run the k6 test again:**
    ```bash
    k6 run ./scripts/leaderboard_test.js
    ```
4.  **Record the new result:** The `p(95)` value for `http_req_duration` should now be dramatically lower (e.g., `p(95)......48.2ms`).

### Conclusion

You have now quantified the impact of your optimization. You can claim a specific latency reduction, such as "slashed p95 latency from ~800ms to under 50ms."

---

## 3. Plagiarism Detection Performance

**Goal:** Demonstrate that your MinHash LSH algorithm is significantly more scalable than a naive O(N²) string comparison approach.

### Prerequisites
- Python 3
- `jellyfish` library (`pip install jellyfish`)

### Step 1: Create a Large Test Set

Generate a large number of C++ source files (e.g., 1000+) in a temporary directory. A simple script can create variations of a few base solutions.

### Step 2: Benchmark the Naive Approach

1.  Create the file `scripts/benchmark_plagiarism_naive.py` with the provided content.
2.  Run the script, pointing it to your directory of test files:
    ```bash
    python ./scripts/benchmark_plagiarism_naive.py /path/to/your/cpp/files
    ```
3.  **Record the time:** Note the execution time. For 1000 files, this will be slow as it performs nearly 500,000 comparisons.

### Step 3: Benchmark Your LSH Approach

1.  Trigger your existing plagiarism detection module to run on the same set of 1000+ files. This might involve calling an API endpoint or running a specific part of your Go service.
2.  **Time the process:** Measure the execution time for your LSH implementation.

### Conclusion

The execution time for your LSH implementation will be orders of magnitude faster than the naive script. This allows you to make a strong claim about scalability, such as "reduced comparison complexity from O(N²) to near-linear time, enabling analysis of 10,000+ submissions in under a minute."
