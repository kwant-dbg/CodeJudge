#ifndef SANDBOX_H
#define SANDBOX_H

#include <string>
#include <memory>

class SecureSandbox
{
public:
    struct SandboxConfig
    {
        std::string chroot_dir;
        std::string user;
        std::string group;
        size_t memory_limit_mb = 256;
        int time_limit_seconds = 2;
        bool enable_network = false;
        bool enable_filesystem_write = false;
    };

    struct SandboxResult
    {
        int exit_code;
        bool timeout;
        bool memory_exceeded;
        bool signal_killed;
        int signal;
        std::string output;
        std::string error;
    };

    SecureSandbox(const SandboxConfig &config);
    ~SecureSandbox();

    SandboxResult execute(const std::string &executable_path, const std::string &input);

private:
    SandboxConfig config_;
    std::string sandbox_root_;

    bool setup_chroot_environment();
    bool setup_namespaces();
    bool setup_seccomp_filter();
    bool setup_cgroups();
    void cleanup();
};

#endif // SANDBOX_H