#include "sandbox.h"
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/resource.h>
#include <sys/mount.h>
#include <sys/prctl.h>
#include <sys/stat.h>
#include <sched.h>
#include <seccomp.h>
#include <pwd.h>
#include <grp.h>
#include <errno.h>
#include <vector>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <cstring>

SecureSandbox::SecureSandbox(const SandboxConfig &config) : config_(config)
{
    sandbox_root_ = "/tmp/sandbox_" + std::to_string(getpid());
    setup_chroot_environment();
}

SecureSandbox::~SecureSandbox()
{
    cleanup();
}

bool SecureSandbox::setup_chroot_environment()
{
    // Create sandbox directory
    std::filesystem::create_directories(sandbox_root_);
    std::filesystem::create_directories(sandbox_root_ + "/tmp");
    std::filesystem::create_directories(sandbox_root_ + "/dev");
    std::filesystem::create_directories(sandbox_root_ + "/proc");
    std::filesystem::create_directories(sandbox_root_ + "/bin");
    std::filesystem::create_directories(sandbox_root_ + "/lib");
    std::filesystem::create_directories(sandbox_root_ + "/lib64");

    // Copy essential binaries and libraries
    // This is a simplified version - in production you'd copy only necessary files
    return true;
}

bool SecureSandbox::setup_seccomp_filter()
{
    scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_KILL);
    if (!ctx)
    {
        return false;
    }

    // Allow essential system calls only
    std::vector<int> allowed_syscalls = {
        SCMP_SYS(read), SCMP_SYS(write), SCMP_SYS(exit), SCMP_SYS(exit_group),
        SCMP_SYS(rt_sigreturn), SCMP_SYS(brk), SCMP_SYS(mmap), SCMP_SYS(munmap),
        SCMP_SYS(mprotect), SCMP_SYS(close), SCMP_SYS(fstat), SCMP_SYS(lseek),
        SCMP_SYS(arch_prctl), SCMP_SYS(access), SCMP_SYS(rt_sigaction),
        SCMP_SYS(rt_sigprocmask), SCMP_SYS(ioctl), SCMP_SYS(readv), SCMP_SYS(writev),
        SCMP_SYS(execve), SCMP_SYS(open), SCMP_SYS(openat), SCMP_SYS(stat),
        SCMP_SYS(newfstatat), SCMP_SYS(getdents64), SCMP_SYS(pread64), SCMP_SYS(pwrite64)};

    for (int syscall : allowed_syscalls)
    {
        if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, syscall, 0) < 0)
        {
            seccomp_release(ctx);
            return false;
        }
    }

    // Block dangerous syscalls explicitly
    std::vector<int> blocked_syscalls = {
        SCMP_SYS(fork), SCMP_SYS(vfork), SCMP_SYS(clone),
        SCMP_SYS(ptrace), SCMP_SYS(kill), SCMP_SYS(socket), SCMP_SYS(connect),
        SCMP_SYS(bind), SCMP_SYS(listen), SCMP_SYS(accept), SCMP_SYS(sendto),
        SCMP_SYS(recvfrom), SCMP_SYS(mount), SCMP_SYS(umount2), SCMP_SYS(chroot)};

    for (int syscall : blocked_syscalls)
    {
        seccomp_rule_add(ctx, SCMP_ACT_KILL, syscall, 0);
    }

    int result = seccomp_load(ctx);
    seccomp_release(ctx);
    return result == 0;
}

bool SecureSandbox::setup_cgroups()
{
    // Create cgroup for memory limiting
    std::string cgroup_path = "/sys/fs/cgroup/memory/sandbox_" + std::to_string(getpid());

    if (mkdir(cgroup_path.c_str(), 0755) == -1 && errno != EEXIST)
    {
        return false;
    }

    // Set memory limit
    std::ofstream memory_limit(cgroup_path + "/memory.limit_in_bytes");
    if (memory_limit.is_open())
    {
        memory_limit << (config_.memory_limit_mb * 1024 * 1024);
        memory_limit.close();
    }

    // Add current process to cgroup
    std::ofstream tasks(cgroup_path + "/tasks");
    if (tasks.is_open())
    {
        tasks << getpid();
        tasks.close();
    }

    return true;
}

SecureSandbox::SandboxResult SecureSandbox::execute(const std::string &executable_path, const std::string &input)
{
    SandboxResult result = {};

    int input_pipe[2];
    int output_pipe[2];
    int error_pipe[2];

    if (pipe(input_pipe) == -1 || pipe(output_pipe) == -1 || pipe(error_pipe) == -1)
    {
        result.exit_code = -1;
        return result;
    }

    pid_t pid = fork();
    if (pid == 0)
    {
        // Child process - set up secure environment

        // Create new namespaces for isolation
        if (unshare(CLONE_NEWPID | CLONE_NEWNET | CLONE_NEWNS | CLONE_NEWUTS | CLONE_NEWIPC) == -1)
        {
            exit(127);
        }

        // Set up resource limits
        struct rlimit time_limit;
        time_limit.rlim_cur = config_.time_limit_seconds;
        time_limit.rlim_max = config_.time_limit_seconds;
        setrlimit(RLIMIT_CPU, &time_limit);

        struct rlimit mem_limit;
        mem_limit.rlim_cur = config_.memory_limit_mb * 1024 * 1024;
        mem_limit.rlim_max = config_.memory_limit_mb * 1024 * 1024;
        setrlimit(RLIMIT_AS, &mem_limit);

        // Limit file descriptors
        struct rlimit fd_limit;
        fd_limit.rlim_cur = 64;
        fd_limit.rlim_max = 64;
        setrlimit(RLIMIT_NOFILE, &fd_limit);

        // Limit number of processes
        struct rlimit proc_limit;
        proc_limit.rlim_cur = 1;
        proc_limit.rlim_max = 1;
        setrlimit(RLIMIT_NPROC, &proc_limit);

        // Change to restricted user if specified
        if (!config_.user.empty())
        {
            struct passwd *pw = getpwnam(config_.user.c_str());
            if (pw)
            {
                setgid(pw->pw_gid);
                setuid(pw->pw_uid);
            }
        }

        // Note: Seccomp filter disabled for now as it interferes with execve
        // TODO: Implement seccomp filter that allows execve or use a different approach
        // setup_seccomp_filter();

        // Disable core dumps
        prctl(PR_SET_DUMPABLE, 0);

        // Set up pipes
        dup2(input_pipe[0], STDIN_FILENO);
        dup2(output_pipe[1], STDOUT_FILENO);
        dup2(error_pipe[1], STDERR_FILENO);

        close(input_pipe[0]);
        close(input_pipe[1]);
        close(output_pipe[0]);
        close(output_pipe[1]);
        close(error_pipe[0]);
        close(error_pipe[1]);

        // Execute the program
        execl(executable_path.c_str(), executable_path.c_str(), (char *)NULL);
        exit(127);
    }
    else if (pid > 0)
    {
        // Parent process
        close(input_pipe[0]);
        close(output_pipe[1]);
        close(error_pipe[1]);

        // Send input
        write(input_pipe[1], input.c_str(), input.length());
        close(input_pipe[1]);

        // Read output
        char buffer[4096];
        ssize_t count;
        while ((count = read(output_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
        {
            buffer[count] = '\0';
            result.output += buffer;
        }
        close(output_pipe[0]);

        // Read error output
        while ((count = read(error_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
        {
            buffer[count] = '\0';
            result.error += buffer;
        }
        close(error_pipe[0]);

        // Wait for child process
        int status;
        waitpid(pid, &status, 0);

        if (WIFSIGNALED(status))
        {
            result.signal_killed = true;
            result.signal = WTERMSIG(status);
            if (WTERMSIG(status) == SIGXCPU)
            {
                result.timeout = true;
            }
        }
        else if (WIFEXITED(status))
        {
            result.exit_code = WEXITSTATUS(status);
        }
    }
    else
    {
        result.exit_code = -1;
    }

    return result;
}

void SecureSandbox::cleanup()
{
    // Clean up sandbox directory
    if (!sandbox_root_.empty())
    {
        std::filesystem::remove_all(sandbox_root_);
    }

    // Clean up cgroup
    std::string cgroup_path = "/sys/fs/cgroup/memory/sandbox_" + std::to_string(getpid());
    rmdir(cgroup_path.c_str());
}