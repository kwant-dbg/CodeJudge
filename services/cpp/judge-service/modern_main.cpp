#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <memory>
#include <filesystem>
#include <chrono>
#include <thread>
#include <exception>

#include <hiredis/hiredis.h>
#include <libpq-fe.h>
#include <nlohmann/json.hpp>

#include "sandbox.h"

using json = nlohmann::json;

// RAII wrapper for PostgreSQL connection
class DatabaseConnection
{
private:
    PGconn *conn_;

public:
    explicit DatabaseConnection(const std::string &connection_string)
    {
        conn_ = PQconnectdb(connection_string.c_str());
        if (PQstatus(conn_) != CONNECTION_OK)
        {
            std::string error = PQerrorMessage(conn_);
            PQfinish(conn_);
            throw std::runtime_error("Database connection failed: " + error);
        }
    }

    ~DatabaseConnection()
    {
        if (conn_)
        {
            PQfinish(conn_);
        }
    }

    // Non-copyable
    DatabaseConnection(const DatabaseConnection &) = delete;
    DatabaseConnection &operator=(const DatabaseConnection &) = delete;

    // Movable
    DatabaseConnection(DatabaseConnection &&other) noexcept : conn_(other.conn_)
    {
        other.conn_ = nullptr;
    }

    DatabaseConnection &operator=(DatabaseConnection &&other) noexcept
    {
        if (this != &other)
        {
            if (conn_)
                PQfinish(conn_);
            conn_ = other.conn_;
            other.conn_ = nullptr;
        }
        return *this;
    }

    PGconn *get() const { return conn_; }

    bool is_valid() const { return conn_ && PQstatus(conn_) == CONNECTION_OK; }
};

// RAII wrapper for Redis connection
class RedisConnection
{
private:
    redisContext *context_;

public:
    explicit RedisConnection(const std::string &host, int port)
    {
        context_ = redisConnect(host.c_str(), port);
        if (!context_ || context_->err)
        {
            std::string error = context_ ? context_->errstr : "Unknown connection error";
            if (context_)
                redisFree(context_);
            throw std::runtime_error("Redis connection failed: " + error);
        }
    }

    ~RedisConnection()
    {
        if (context_)
        {
            redisFree(context_);
        }
    }

    // Non-copyable
    RedisConnection(const RedisConnection &) = delete;
    RedisConnection &operator=(const RedisConnection &) = delete;

    // Movable
    RedisConnection(RedisConnection &&other) noexcept : context_(other.context_)
    {
        other.context_ = nullptr;
    }

    RedisConnection &operator=(RedisConnection &&other) noexcept
    {
        if (this != &other)
        {
            if (context_)
                redisFree(context_);
            context_ = other.context_;
            other.context_ = nullptr;
        }
        return *this;
    }

    redisContext *get() const { return context_; }

    bool is_valid() const { return context_ && !context_->err; }
};

struct TestCase
{
    int id;
    std::string input;
    std::string expected_output;
};

struct Submission
{
    int id;
    int problem_id;
    std::string source_code;
    std::vector<TestCase> test_cases;
};

class ModernJudgeService
{
private:
    std::unique_ptr<DatabaseConnection> db_;
    std::unique_ptr<RedisConnection> redis_;
    SecureSandbox::SandboxConfig sandbox_config_;

public:
    ModernJudgeService(const std::string &db_url, const std::string &redis_host, int redis_port)
    {
        // Initialize database connection
        db_ = std::make_unique<DatabaseConnection>(db_url);

        // Initialize Redis connection
        redis_ = std::make_unique<RedisConnection>(redis_host, redis_port);

        // Configure secure sandbox
        sandbox_config_.memory_limit_mb = 256;
        sandbox_config_.time_limit_seconds = 2;
        sandbox_config_.enable_network = false;
        sandbox_config_.enable_filesystem_write = false;
        sandbox_config_.user = "nobody"; // Run as restricted user
    }

    std::vector<TestCase> fetch_test_cases(int problem_id)
    {
        const char *query = "SELECT id, input, output FROM test_cases WHERE problem_id = $1";
        const char *param_values[] = {std::to_string(problem_id).c_str()};

        PGresult *result = PQexecParams(db_->get(), query, 1, nullptr, param_values, nullptr, nullptr, 0);

        if (PQresultStatus(result) != PGRES_TUPLES_OK)
        {
            std::string error = PQerrorMessage(db_->get());
            PQclear(result);
            throw std::runtime_error("Failed to fetch test cases: " + error);
        }

        std::vector<TestCase> test_cases;
        int rows = PQntuples(result);

        for (int i = 0; i < rows; i++)
        {
            TestCase tc;
            tc.id = std::stoi(PQgetvalue(result, i, 0));
            tc.input = PQgetvalue(result, i, 1);
            tc.expected_output = PQgetvalue(result, i, 2);
            test_cases.push_back(tc);
        }

        PQclear(result);
        return test_cases;
    }

    Submission fetch_submission(int submission_id)
    {
        const char *query = "SELECT id, problem_id, source_code FROM submissions WHERE id = $1";
        const char *param_values[] = {std::to_string(submission_id).c_str()};

        PGresult *result = PQexecParams(db_->get(), query, 1, nullptr, param_values, nullptr, nullptr, 0);

        if (PQresultStatus(result) != PGRES_TUPLES_OK || PQntuples(result) == 0)
        {
            PQclear(result);
            throw std::runtime_error("Submission not found");
        }

        Submission submission;
        submission.id = std::stoi(PQgetvalue(result, 0, 0));
        submission.problem_id = std::stoi(PQgetvalue(result, 0, 1));
        submission.source_code = PQgetvalue(result, 0, 2);

        PQclear(result);

        // Fetch test cases
        submission.test_cases = fetch_test_cases(submission.problem_id);

        return submission;
    }

    bool compile_submission(const Submission &submission, const std::string &executable_path)
    {
        // Create temporary source file
        std::string source_path = "/tmp/submission_" + std::to_string(submission.id) + ".cpp";

        std::ofstream source_file(source_path);
        if (!source_file)
        {
            return false;
        }
        source_file << submission.source_code;
        source_file.close();

        // Compile with timeout and resource limits
        SecureSandbox compiler_sandbox(sandbox_config_);
        SecureSandbox::SandboxResult result = compiler_sandbox.execute(
            "/usr/bin/g++",
            source_path + " -o " + executable_path + " -std=c++17 -O2");

        // Clean up source file
        std::filesystem::remove(source_path);

        return result.exit_code == 0;
    }

    std::string determine_verdict(const SecureSandbox::SandboxResult &result, const std::string &expected_output)
    {
        if (result.timeout)
        {
            return "Time Limit Exceeded";
        }

        if (result.memory_exceeded)
        {
            return "Memory Limit Exceeded";
        }

        if (result.signal_killed || result.exit_code != 0)
        {
            return "Runtime Error";
        }

        // Normalize output for comparison
        std::string actual_output = result.output;
        std::string expected = expected_output;

        // Remove trailing whitespace
        actual_output.erase(actual_output.find_last_not_of(" \n\r\t") + 1);
        expected.erase(expected.find_last_not_of(" \n\r\t") + 1);

        return (actual_output == expected) ? "Accepted" : "Wrong Answer";
    }

    std::string judge_submission(const Submission &submission)
    {
        std::string executable_path = "/tmp/submission_" + std::to_string(submission.id);

        try
        {
            // Compile the submission
            if (!compile_submission(submission, executable_path))
            {
                return "Compilation Error";
            }

            // Run against test cases
            SecureSandbox sandbox(sandbox_config_);

            for (const auto &test_case : submission.test_cases)
            {
                auto result = sandbox.execute(executable_path, test_case.input);
                std::string verdict = determine_verdict(result, test_case.expected_output);

                if (verdict != "Accepted")
                {
                    std::filesystem::remove(executable_path);
                    return verdict;
                }
            }

            std::filesystem::remove(executable_path);
            return "Accepted";
        }
        catch (const std::exception &e)
        {
            std::filesystem::remove(executable_path);
            std::cerr << "Judge error: " << e.what() << std::endl;
            return "Judge Error";
        }
    }

    void update_verdict(int submission_id, const std::string &verdict)
    {
        const char *query = "UPDATE submissions SET verdict = $1, judged_at = NOW() WHERE id = $2";
        const char *param_values[] = {verdict.c_str(), std::to_string(submission_id).c_str()};

        PGresult *result = PQexecParams(db_->get(), query, 2, nullptr, param_values, nullptr, nullptr, 0);

        if (PQresultStatus(result) != PGRES_COMMAND_OK)
        {
            std::string error = PQerrorMessage(db_->get());
            PQclear(result);
            throw std::runtime_error("Failed to update verdict: " + error);
        }

        PQclear(result);
    }

    void process_submission_queue()
    {
        while (true)
        {
            try
            {
                // Block and wait for submission from Redis queue
                redisReply *reply = (redisReply *)redisCommand(redis_->get(), "BRPOP submission_queue 0");

                if (!reply || reply->type != REDIS_REPLY_ARRAY || reply->elements != 2)
                {
                    if (reply)
                        freeReplyObject(reply);
                    continue;
                }

                int submission_id = std::stoi(reply->element[1]->str);
                freeReplyObject(reply);

                std::cout << "Processing submission " << submission_id << std::endl;

                // Fetch and judge submission
                Submission submission = fetch_submission(submission_id);
                std::string verdict = judge_submission(submission);

                // Update database with verdict
                update_verdict(submission_id, verdict);

                std::cout << "Submission " << submission_id << " judged: " << verdict << std::endl;
            }
            catch (const std::exception &e)
            {
                std::cerr << "Error processing submission: " << e.what() << std::endl;
                std::this_thread::sleep_for(std::chrono::seconds(1));
            }
        }
    }
};

int main()
{
    try
    {
        // Get configuration from environment
        const char *db_url = std::getenv("DATABASE_URL");
        const char *redis_host = std::getenv("REDIS_HOST");
        const char *redis_port_str = std::getenv("REDIS_PORT");

        if (!db_url)
            db_url = "postgresql://localhost/codejudge";
        if (!redis_host)
            redis_host = "localhost";
        int redis_port = redis_port_str ? std::stoi(redis_port_str) : 6379;

        std::cout << "Starting Modern Judge Service..." << std::endl;
        std::cout << "Database: " << db_url << std::endl;
        std::cout << "Redis: " << redis_host << ":" << redis_port << std::endl;

        ModernJudgeService judge(db_url, redis_host, redis_port);
        judge.process_submission_queue();
    }
    catch (const std::exception &e)
    {
        std::cerr << "Fatal error: " << e.what() << std::endl;
        return 1;
    }

    return 0;
}