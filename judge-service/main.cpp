#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/resource.h>
#include <hiredis/hiredis.h>

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

std::string run_code(const std::string& executable_path, const std::string& input_path) {
    int pipe_fd[2];
    if (pipe(pipe_fd) == -1) return "RUNTIME_ERROR";

    pid_t pid = fork();
    if (pid == 0) {
        set_limits();
        freopen(input_path.c_str(), "r", stdin);
        dup2(pipe_fd[1], STDOUT_FILENO);
        close(pipe_fd[0]);
        close(pipe_fd[1]);
        execl(executable_path.c_str(), executable_path.c_str(), (char*)NULL);
        exit(127);
    } else if (pid > 0) {
        close(pipe_fd[1]);
        std::string output = "";
        char buffer[1024];
        ssize_t count;

        while ((count = read(pipe_fd[0], buffer, sizeof(buffer))) > 0) {
            output.append(buffer, count);
        }
        close(pipe_fd[0]);

        int status;
        waitpid(pid, &status, 0);

        if (WIFEXITED(status) && WEXITSTATUS(status) == 0) return output;
        if (WIFSIGNALED(status) && WTERMSIG(status) == SIGXCPU) return "TIME_LIMIT_EXCEEDED";
        
        return "RUNTIME_ERROR";
    }
    return "JUDGE_ERROR";
}

void process_submission(const std::string& submission_id) {
    std::cout << "Processing submission ID: " << submission_id << std::endl;

    std::string source_code = R"(
#include <iostream>
int main() {
    long long a, b;
    std::cin >> a >> b;
    std::cout << a + b << std::endl;
    return 0;
}
    )";
    std::string test_input = "5 10\n";
    std::string expected_output = "15\n";

    std::string source_path = "/app/submissions/" + submission_id + ".cpp";
    std::ofstream source_file(source_path);
    source_file << source_code;
    source_file.close();

    std::string input_path = "/app/submissions/" + submission_id + "_input.txt";
    std::ofstream input_file(input_path);
    input_file << test_input;
    input_file.close();

    std::string executable_path = "/app/submissions/" + submission_id;
    if (!compile_code(source_path, executable_path)) {
        std::cout << "Verdict for " << submission_id << ": Compilation Error" << std::endl;
        return;
    }

    std::string user_output = run_code(executable_path, input_path);

    if (user_output == expected_output) {
        std::cout << "Verdict for " << submission_id << ": Accepted" << std::endl;
    } else if (user_output == "TIME_LIMIT_EXCEEDED") {
        std::cout << "Verdict for " << submission_id << ": Time Limit Exceeded" << std::endl;
    } else {
        std::cout << "Verdict for " << submission_id << ": Wrong Answer" << std::endl;
    }

    remove(source_path.c_str());
    remove(input_path.c_str());
    remove(executable_path.c_str());
}

int main() {
    redisContext *redis_c = redisConnect("redis", 6379);
    if (redis_c == NULL || redis_c->err) {
        if (redis_c) {
            std::cerr << "Redis connection error: " << redis_c->errstr << std::endl;
            redisFree(redis_c);
        } else {
            std::cerr << "Can't allocate redis context" << std::endl;
        }
        return 1;
    }

    std::cout << "Judge Service Started. Waiting for submissions..." << std::endl;

    while (true) {
        redisReply *reply;
        reply = (redisReply*)redisCommand(redis_c, "BLPOP submission_queue 0");
        if (reply->type == REDIS_REPLY_ARRAY && reply->elements == 2) {
            process_submission(reply->element[1]->str);
        }
        freeReplyObject(reply);
    }

    redisFree(redis_c);
    return 0;
}
