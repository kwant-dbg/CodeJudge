#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/resource.h>
#include <hiredis/hiredis.h>
#include <libpq-fe.h>
#include <cstdio>
#include <stdexcept>
#include <filesystem>

struct TestCase
{
    std::string input;
    std::string output;
};

struct RedisConfig
{
    std::string host = "redis";
    int port = 6379;
    std::string password;
};

static std::string rtrim_copy(const std::string &s)
{
    size_t end = s.find_last_not_of(" \n\r\t");
    return (end == std::string::npos) ? "" : s.substr(0, end + 1);
}

static std::string verdict_from_output(const std::string &run_out, const std::string &expected)
{
    if (run_out == "TIME_LIMIT_EXCEEDED")
        return "Time Limit Exceeded";
    if (run_out == "RUNTIME_ERROR" || run_out == "JUDGE_ERROR")
        return "Runtime Error";
    return rtrim_copy(run_out) == rtrim_copy(expected) ? "Accepted" : "Wrong Answer";
}

RedisConfig parse_redis_url(const std::string &url)
{
    RedisConfig config;
    if (url.empty())
    {
        return config;
    }

    std::string working = url;
    auto scheme_pos = working.find("://");
    if (scheme_pos != std::string::npos)
    {
        working = working.substr(scheme_pos + 3);
    }

    auto at_pos = working.find('@');
    if (at_pos != std::string::npos)
    {
        std::string credentials = working.substr(0, at_pos);
        working = working.substr(at_pos + 1);

        if (!credentials.empty())
        {
            if (credentials.front() == ':')
            {
                config.password = credentials.substr(1);
            }
            else
            {
                auto colon_pos = credentials.find(':');
                if (colon_pos != std::string::npos)
                {
                    config.password = credentials.substr(colon_pos + 1);
                }
                else
                {
                    config.password = credentials;
                }
            }
        }
    }

    auto end_host = working.find_first_of("/?");
    std::string host_port = working.substr(0, end_host);
    if (!host_port.empty())
    {
        auto colon_pos = host_port.find(':');
        if (colon_pos != std::string::npos)
        {
            config.host = host_port.substr(0, colon_pos);
            try
            {
                config.port = std::stoi(host_port.substr(colon_pos + 1));
            }
            catch (const std::exception &)
            {
                config.port = 6379;
            }
        }
        else
        {
            config.host = host_port;
        }
    }

    return config;
}

bool fetch_source_code(PGconn *db_conn, const std::string &submission_id, std::string &source_code)
{
    const char *paramValues[1] = {submission_id.c_str()};
    PGresult *res = PQexecParams(db_conn, "SELECT source_code FROM submissions WHERE id = $1", 1, NULL, paramValues, NULL, NULL, 0);

    if (PQresultStatus(res) != PGRES_TUPLES_OK || PQntuples(res) != 1)
    {
        PQclear(res);
        return false;
    }

    source_code = PQgetvalue(res, 0, 0);
    PQclear(res);
    return true;
}

bool write_source_to_disk(const std::string &path, const std::string &contents)
{
    std::ofstream out(path);
    if (!out.is_open())
    {
        return false;
    }
    out << contents;
    out.close();
    return out.good();
}

void set_limits()
{
    struct rlimit time_limit;
    time_limit.rlim_cur = 2;
    time_limit.rlim_max = 2;
    setrlimit(RLIMIT_CPU, &time_limit);

    struct rlimit mem_limit;
    mem_limit.rlim_cur = 256 * 1024 * 1024;
    mem_limit.rlim_max = 256 * 1024 * 1024;
    setrlimit(RLIMIT_AS, &mem_limit);
}

bool compile_code(const std::string &source_path, const std::string &executable_path)
{
    pid_t pid = fork();
    if (pid == 0)
    {
        execlp("g++", "g++", source_path.c_str(), "-o", executable_path.c_str(), "-std=c++17", (char *)NULL);
        exit(1);
    }
    else if (pid > 0)
    {
        int status;
        waitpid(pid, &status, 0);
        return WIFEXITED(status) && WEXITSTATUS(status) == 0;
    }
    return false;
}

std::string run_code(const std::string &executable_path, const std::string &input)
{
    int input_pipe[2];
    int output_pipe[2];
    if (pipe(input_pipe) == -1 || pipe(output_pipe) == -1)
        return "JUDGE_ERROR";

    pid_t pid = fork();
    if (pid == 0)
    {
        set_limits();

        dup2(input_pipe[0], STDIN_FILENO);
        dup2(output_pipe[1], STDOUT_FILENO);

        close(input_pipe[0]);
        close(input_pipe[1]);
        close(output_pipe[0]);
        close(output_pipe[1]);

        execlp(executable_path.c_str(), executable_path.c_str(), (char *)NULL);
        exit(127);
    }
    else if (pid > 0)
    {
        close(input_pipe[0]);
        close(output_pipe[1]);

        write(input_pipe[1], input.c_str(), input.length());
        close(input_pipe[1]);

        std::string output = "";
        char buffer[1024];
        ssize_t count;
        while ((count = read(output_pipe[0], buffer, sizeof(buffer))) > 0)
        {
            output.append(buffer, count);
        }
        close(output_pipe[0]);

        int status;
        waitpid(pid, &status, 0);

        if (WIFSIGNALED(status))
        {
            if (WTERMSIG(status) == SIGXCPU)
                return "TIME_LIMIT_EXCEEDED";
            return "RUNTIME_ERROR";
        }
        if (WIFEXITED(status) && WEXITSTATUS(status) == 0)
        {
            return output;
        }
        return "RUNTIME_ERROR";
    }
    return "JUDGE_ERROR";
}

void update_verdict(PGconn *db_conn, const std::string &submission_id, const std::string &verdict)
{
    std::string query = "UPDATE submissions SET verdict = $1 WHERE id = $2";
    const char *paramValues[2] = {verdict.c_str(), submission_id.c_str()};
    PGresult *res = PQexecParams(db_conn, query.c_str(), 2, NULL, paramValues, NULL, NULL, 0);
    if (PQresultStatus(res) != PGRES_COMMAND_OK)
    {
        fprintf(stderr, "UPDATE failed: %s\n", PQerrorMessage(db_conn));
    }
    PQclear(res);
}

int get_problem_id(PGconn *db_conn, const std::string &submission_id)
{
    std::string query = "SELECT problem_id FROM submissions WHERE id = $1";
    const char *paramValues[1] = {submission_id.c_str()};
    PGresult *res = PQexecParams(db_conn, query.c_str(), 1, NULL, paramValues, NULL, NULL, 0);

    if (PQresultStatus(res) != PGRES_TUPLES_OK || PQntuples(res) != 1)
    {
        PQclear(res);
        return -1;
    }
    int problem_id = std::stoi(PQgetvalue(res, 0, 0));
    PQclear(res);
    return problem_id;
}

std::vector<TestCase> get_test_cases(PGconn *db_conn, int problem_id)
{
    std::vector<TestCase> test_cases;
    std::string query = "SELECT input, output FROM test_cases WHERE problem_id = $1";
    std::string problem_id_str = std::to_string(problem_id);
    const char *paramValues[1] = {problem_id_str.c_str()};
    PGresult *res = PQexecParams(db_conn, query.c_str(), 1, NULL, paramValues, NULL, NULL, 0);

    if (PQresultStatus(res) == PGRES_TUPLES_OK)
    {
        for (int i = 0; i < PQntuples(res); i++)
        {
            test_cases.push_back({PQgetvalue(res, i, 0),
                                  PQgetvalue(res, i, 1)});
        }
    }
    PQclear(res);
    return test_cases;
}

void process_submission(const std::string &submission_id, PGconn *db_conn)
{
    std::cout << "Processing submission ID: " << submission_id << std::endl;
    std::string workdir = getenv("SUBMISSION_WORKDIR") ? getenv("SUBMISSION_WORKDIR") : "/tmp/codejudge-submissions";
    try
    {
        std::filesystem::create_directories(workdir);
    }
    catch (const std::exception &ex)
    {
        std::cerr << "Failed to create work directory: " << ex.what() << std::endl;
        update_verdict(db_conn, submission_id, "Judge Error: Storage unavailable");
        return;
    }

    std::string source_path = workdir + "/" + submission_id + ".cpp";
    std::string executable_path = workdir + "/" + submission_id;

    auto cleanup_files = [&]()
    {
        std::remove(source_path.c_str());
        std::remove(executable_path.c_str());
    };

    auto finish = [&](const std::string &v)
    {
        update_verdict(db_conn, submission_id, v);
        cleanup_files();
    };

    std::string source_code;
    if (!fetch_source_code(db_conn, submission_id, source_code))
    {
        finish("Judge Error: Source not found");
        return;
    }

    if (!write_source_to_disk(source_path, source_code))
    {
        finish("Judge Error: Write failure");
        return;
    }

    if (!compile_code(source_path, executable_path))
    {
        std::cout << "Verdict for " << submission_id << ": Compilation Error" << std::endl;
        finish("Compilation Error");
        return;
    }

    int problem_id = get_problem_id(db_conn, submission_id);
    if (problem_id == -1)
    {
        finish("Judge Error: Problem not found");
        return;
    }

    std::vector<TestCase> test_cases = get_test_cases(db_conn, problem_id);
    if (test_cases.empty())
    {
        finish("Judge Error: No test cases");
        return;
    }

    std::string final_verdict = "Accepted";
    for (const auto &tc : test_cases)
    {
        std::string verdict = verdict_from_output(run_code(executable_path, tc.input), tc.output);
        if (verdict != "Accepted")
        {
            final_verdict = verdict;
            break;
        }
    }

    std::cout << "Verdict for " << submission_id << ": " << final_verdict << std::endl;
    update_verdict(db_conn, submission_id, final_verdict);

    cleanup_files();
}

int main()
{
    std::string redis_url_env = getenv("REDIS_URL") ? getenv("REDIS_URL") : "redis:6379";
    RedisConfig redis_cfg = parse_redis_url(redis_url_env);
    redisContext *redis_c = redisConnect(redis_cfg.host.c_str(), redis_cfg.port);
    if (redis_c == NULL || redis_c->err)
    {
        if (redis_c)
        {
            std::cerr << "Redis connection error: " << redis_c->errstr << std::endl;
            redisFree(redis_c);
        }
        else
        {
            std::cerr << "Can't allocate redis context" << std::endl;
        }
        return 1;
    }

    if (!redis_cfg.password.empty())
    {
        redisReply *auth_reply = (redisReply *)redisCommand(redis_c, "AUTH %s", redis_cfg.password.c_str());
        if (auth_reply == NULL)
        {
            std::cerr << "Redis AUTH failed" << std::endl;
            redisFree(redis_c);
            return 1;
        }
        if (auth_reply->type == REDIS_REPLY_ERROR)
        {
            std::cerr << "Redis AUTH error: " << auth_reply->str << std::endl;
            freeReplyObject(auth_reply);
            redisFree(redis_c);
            return 1;
        }
        freeReplyObject(auth_reply);
    }

    const char *db_url = getenv("DATABASE_URL");
    if (db_url == NULL)
    {
        std::cerr << "DATABASE_URL not set" << std::endl;
        return 1;
    }
    PGconn *db_conn = PQconnectdb(db_url);
    if (PQstatus(db_conn) != CONNECTION_OK)
    {
        fprintf(stderr, "Connection to database failed: %s\n", PQerrorMessage(db_conn));
        PQfinish(db_conn);
        return 1;
    }

    std::cout << "Judge Service Started. Waiting for submissions..." << std::endl;

    while (true)
    {
        redisReply *reply;
        reply = (redisReply *)redisCommand(redis_c, "BLPOP submission_queue 0");
        if (reply != NULL)
        {
            if (reply->type == REDIS_REPLY_ARRAY && reply->elements == 2)
            {
                process_submission(reply->element[1]->str, db_conn);
            }
            freeReplyObject(reply);
        }
    }

    redisFree(redis_c);
    PQfinish(db_conn);
    return 0;
}
