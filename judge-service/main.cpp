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

struct TestCase {
    std::string input;
    std::string output;
};

void set_limits() {
    struct rlimit time_limit;
    time_limit.rlim_cur = 2;
    time_limit.rlim_max = 2;
    setrlimit(RLIMIT_CPU, &time_limit);

    struct rlimit mem_limit;
    mem_limit.rlim_cur = 256 * 1024 * 1024;
    mem_limit.rlim_max = 256 * 1024 * 1024;
    setrlimit(RLIMIT_AS, &mem_limit);
}

bool compile_code(const std::string& source_path, const std::string& executable_path) {
    pid_t pid = fork();
    if (pid == 0) {
        execlp("g++", "g++", source_path.c_str(), "-o", executable_path.c_str(), "-std=c++17", (char*)NULL);
        exit(1);
    } else if (pid > 0) {
        int status;
        waitpid(pid, &status, 0);
        return WIFEXITED(status) && WEXITSTATUS(status) == 0;
    }
    return false;
}

std::string run_code(const std::string& executable_path, const std::string& input) {
    int input_pipe[2];
    int output_pipe[2];
    if (pipe(input_pipe) == -1 || pipe(output_pipe) == -1) return "JUDGE_ERROR";

    pid_t pid = fork();
    if (pid == 0) {
        set_limits();
        
        dup2(input_pipe[0], STDIN_FILENO);
        dup2(output_pipe[1], STDOUT_FILENO);

        close(input_pipe[0]);
        close(input_pipe[1]);
        close(output_pipe[0]);
        close(output_pipe[1]);
        
        execl(executable_path.c_str(), executable_path.c_str(), (char*)NULL);
        exit(127);
    } else if (pid > 0) {
        close(input_pipe[0]);
        close(output_pipe[1]);

        write(input_pipe[1], input.c_str(), input.length());
        close(input_pipe[1]);

        std::string output = "";
        char buffer[1024];
        ssize_t count;
        while ((count = read(output_pipe[0], buffer, sizeof(buffer))) > 0) {
            output.append(buffer, count);
        }
        close(output_pipe[0]);

        int status;
        waitpid(pid, &status, 0);

        if (WIFSIGNALED(status)) {
            if (WTERMSIG(status) == SIGXCPU) return "TIME_LIMIT_EXCEEDED";
            return "RUNTIME_ERROR";
        }
        if (WIFEXITED(status) && WEXITSTATUS(status) == 0) {
             return output;
        }
        return "RUNTIME_ERROR";
    }
    return "JUDGE_ERROR";
}

void update_verdict(PGconn* db_conn, const std::string& submission_id, const std::string& verdict) {
    std::string query = "UPDATE submissions SET verdict = $1 WHERE id = $2";
    const char* paramValues[2] = {verdict.c_str(), submission_id.c_str()};
    PGresult *res = PQexecParams(db_conn, query.c_str(), 2, NULL, paramValues, NULL, NULL, 0);
    if (PQresultStatus(res) != PGRES_COMMAND_OK) {
        fprintf(stderr, "UPDATE failed: %s\n", PQerrorMessage(db_conn));
    }
    PQclear(res);
}

int get_problem_id(PGconn* db_conn, const std::string& submission_id) {
    std::string query = "SELECT problem_id FROM submissions WHERE id = $1";
    const char* paramValues[1] = {submission_id.c_str()};
    PGresult *res = PQexecParams(db_conn, query.c_str(), 1, NULL, paramValues, NULL, NULL, 0);

    if (PQresultStatus(res) != PGRES_TUPLES_OK || PQntuples(res) != 1) {
        PQclear(res);
        return -1;
    }
    int problem_id = std::stoi(PQgetvalue(res, 0, 0));
    PQclear(res);
    return problem_id;
}

std::vector<TestCase> get_test_cases(PGconn* db_conn, int problem_id) {
    std::vector<TestCase> test_cases;
    std::string query = "SELECT input, output FROM test_cases WHERE problem_id = $1";
    std::string problem_id_str = std::to_string(problem_id);
    const char* paramValues[1] = {problem_id_str.c_str()};
    PGresult *res = PQexecParams(db_conn, query.c_str(), 1, NULL, paramValues, NULL, NULL, 0);

    if (PQresultStatus(res) == PGRES_TUPLES_OK) {
        for (int i = 0; i < PQntuples(res); i++) {
            test_cases.push_back({
                PQgetvalue(res, i, 0),
                PQgetvalue(res, i, 1)
            });
        }
    }
    PQclear(res);
    return test_cases;
}

void process_submission(const std::string& submission_id, PGconn* db_conn) {
    std::cout << "Processing submission ID: " << submission_id << std::endl;
    
    std::string source_path = "/app/submissions/" + submission_id + ".cpp";
    std::string executable_path = "/app/submissions/" + submission_id;

    if (!compile_code(source_path, executable_path)) {
        std::cout << "Verdict for " << submission_id << ": Compilation Error" << std::endl;
        update_verdict(db_conn, submission_id, "Compilation Error");
        return;
    }

    int problem_id = get_problem_id(db_conn, submission_id);
    if (problem_id == -1) {
        update_verdict(db_conn, submission_id, "Judge Error: Problem not found");
        remove(executable_path.c_str());
        return;
    }

    std::vector<TestCase> test_cases = get_test_cases(db_conn, problem_id);
    if (test_cases.empty()) {
        update_verdict(db_conn, submission_id, "Judge Error: No test cases");
        remove(executable_path.c_str());
        return;
    }
    
    std::string final_verdict = "Accepted";
    for (const auto& tc : test_cases) {
        std::string user_output_str = run_code(executable_path, tc.input);
        
        std::string verdict;
        if (user_output_str == "TIME_LIMIT_EXCEEDED") {
            verdict = "Time Limit Exceeded";
        } else if (user_output_str == "RUNTIME_ERROR" || user_output_str == "JUDGE_ERROR") {
            verdict = "Runtime Error";
        } else {
            // Trim trailing whitespace from user output
            size_t end = user_output_str.find_last_not_of(" \n\r\t");
            std::string trimmed_output = (end == std::string::npos) ? "" : user_output_str.substr(0, end + 1);
            
            // Trim trailing whitespace from expected output
            end = tc.output.find_last_not_of(" \n\r\t");
            std::string trimmed_expected = (end == std::string::npos) ? "" : tc.output.substr(0, end + 1);

            if (trimmed_output == trimmed_expected) {
                verdict = "Accepted";
            } else {
                verdict = "Wrong Answer";
            }
        }

        if (verdict != "Accepted") {
            final_verdict = verdict;
            break; // Stop on first failed test case
        }
    }
    
    std::cout << "Verdict for " << submission_id << ": " << final_verdict << std::endl;
    update_verdict(db_conn, submission_id, final_verdict);
    
    remove(executable_path.c_str());
}

int main() {
    std::string redis_url_str = getenv("REDIS_URL") ? getenv("REDIS_URL") : "redis:6379";
    std::string redis_host = "redis";
    int redis_port = 6379;

    size_t colon_pos = redis_url_str.find(':');
    if (colon_pos != std::string::npos) {
        redis_host = redis_url_str.substr(0, colon_pos);
        redis_port = std::stoi(redis_url_str.substr(colon_pos + 1));
    }

    redisContext *redis_c = redisConnect(redis_host.c_str(), redis_port);
    if (redis_c == NULL || redis_c->err) {
        if (redis_c) {
            std::cerr << "Redis connection error: " << redis_c->errstr << std::endl;
            redisFree(redis_c);
        } else {
            std::cerr << "Can't allocate redis context" << std::endl;
        }
        return 1;
    }

    const char* db_url = getenv("DATABASE_URL");
    if (db_url == NULL) {
        std::cerr << "DATABASE_URL not set" << std::endl;
        return 1;
    }
    PGconn *db_conn = PQconnectdb(db_url);
    if (PQstatus(db_conn) != CONNECTION_OK) {
        fprintf(stderr, "Connection to database failed: %s\n", PQerrorMessage(db_conn));
        PQfinish(db_conn);
        return 1;
    }

    std::cout << "Judge Service Started. Waiting for submissions..." << std::endl;

    while (true) {
        redisReply *reply;
        reply = (redisReply*)redisCommand(redis_c, "BLPOP submission_queue 0");
        if (reply != NULL && reply->type == REDIS_REPLY_ARRAY && reply->elements == 2) {
            process_submission(reply->element[1]->str, db_conn);
        }
        if (reply != NULL) {
            freeReplyObject(reply);
        }
    }

    redisFree(redis_c);
    PQfinish(db_conn);
    return 0;
}
