//go:build linux

package arapuca

/*
#include <arapuca.h>
#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>

// Count argc and build argv from /proc/self/cmdline. Returns
// pointers into static buffers; nothing to free. Returns 0 on
// failure.
static char cmdline_buf[65536];
static char *cmdline_argv[4096];

static int read_cmdline(int *out_argc, char ***out_argv) {
    int fd = open("/proc/self/cmdline", O_RDONLY);
    if (fd < 0)
        return 0;
    ssize_t total = 0;
    while (total < (ssize_t)sizeof(cmdline_buf) - 1) {
        ssize_t n = read(fd, cmdline_buf + total,
                         sizeof(cmdline_buf) - 1 - total);
        if (n < 0) {
            if (errno == EINTR)
                continue;
            break;
        }
        if (n == 0)
            break;
        total += n;
    }
    close(fd);
    if (total <= 0)
        return 0;
    cmdline_buf[total] = '\0';

    int argc = 0;
    char *p = cmdline_buf;
    char *end = cmdline_buf + total;
    while (p < end && argc < 4095) {
        cmdline_argv[argc++] = p;
        p += strlen(p) + 1;
    }
    cmdline_argv[argc] = NULL;
    *out_argc = argc;
    *out_argv = cmdline_argv;
    return 1;
}

// Runs before Go's runtime starts. ALL checks are done in C so
// that Rust code is NEVER entered on normal startup — zero
// allocations, zero panic risk, zero cost.
//
// Two-factor gate:
//   1. ARAPUCA_WRAPPER=1 in the environment
//   2. argv[1] == "--"  (the wrapper separator)
//
// Both conditions are only true when the library re-exec'd this
// binary as a sandbox trampoline. Accidental ARAPUCA_WRAPPER=1
// in the ambient environment (without the argv pattern) is
// harmless — the constructor is a no-op.
//
// When both conditions match, enters Rust which applies
// Landlock/seccomp and execve-s into the handler. Never returns.
__attribute__((constructor))
static void arapuca_go_preinit(void) {
    const char *val = getenv("ARAPUCA_WRAPPER");
    if (!val || val[0] != '1' || val[1] != '\0')
        return;

    int argc;
    char **argv;
    if (!read_cmdline(&argc, &argv))
        return;
    if (argc < 2 || strcmp(argv[1], "--") != 0)
        return;

    arapuca_handle_selfexec_if_wrapper(argc, (const char *const *)argv);
    // Should never reach here. If it does, _exit fail-closed.
    _exit(126);
}
*/
import "C"
