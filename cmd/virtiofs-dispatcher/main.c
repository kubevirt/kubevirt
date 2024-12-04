// SPDX-License-Identifier: Apache-2.0

#define _GNU_SOURCE
#include <getopt.h>
#include <stdio.h>
#include <stdarg.h>
#include <string.h>
#include <stdlib.h>
#include <stdbool.h>
#include <limits.h>
#include <unistd.h>
#include <time.h>
#include <errno.h>
#include <sched.h>
#include <sys/syscall.h>
#include <fcntl.h>
#include <sys/file.h>

struct arguments {
    char socket_flag[PATH_MAX + 1];
    char shareddir_flag[PATH_MAX + 1];
    int pid;
};

struct option long_options[] = {
    {"socket-path", required_argument, 0, 's'},
    {"shared-dir", required_argument, 0, 'd'},
    {"pid", required_argument, 0, 'p'},
    {0, 0, 0, 0}
};

void usage()
{
    printf("virtiofsd dispatcher\n"
           "Usage:\n"
           "\t-p, --pid:\t\tPid of the container\n"
           "\t-d  --shared-dir\tShared directory flag for virtiofs\n"
           "\t-s  --socket-path\tSocket path flag for virtiofs\n"
          );
}

#define FMT_SZ 50
void error_log(const char *format, ...)
{
    char time_fmt[FMT_SZ + 1] = {0};
    time_t ltime = time(NULL);
    strftime(time_fmt, FMT_SZ, "%b %d %H:%M:%S ", localtime(&ltime));
    fprintf(stderr, "%s", time_fmt);

    fprintf(stderr, "error: ");

    va_list arglist;
    va_start(arglist, format);
    vfprintf(stderr, format, arglist);
    va_end(arglist);
}

int parse_arguments(int argc, char **argv, struct arguments *args)
{
    while(1) {
        int c = getopt_long(argc, argv, "p:d:s:", long_options, NULL);
        if (c == -1) break;

        switch (c) {
            case 'p':
                args->pid = atoi(optarg);
                break;
            case 'd':
                strncpy(args->shareddir_flag, optarg, PATH_MAX);
                break;
            case 's':
                strncpy(args->socket_flag, optarg, PATH_MAX);
                break;
            case '?':  // fallthrough
            default:
                return -1;
        }
    }

    if (args->pid < 1) {
        error_log("pid needs to be set\n");
        return -1;
    }

    return 0;
}

int do_move_into_cgroup(char *cgroup_entry)
{
    // We must remove the trailing '\n'
    cgroup_entry[strlen(cgroup_entry) - 1] = '\0';

    char syspath[PATH_MAX + 1] = {0};
    if (cgroup_entry[4] == '\0') {
        // Handling "0::/"
        strncpy(syspath, "/sys/fs/cgroup/cgroup.procs", PATH_MAX);
    } else {
        // Let's skip the first 4 characters "0::/" to get the relative path
        snprintf(syspath, PATH_MAX, "/sys/fs/cgroup/%s/cgroup.procs", (cgroup_entry + 4));
    }

    fprintf(stderr, "moving the process into the cgroup: %s\n", syspath);

    FILE *fptr = fopen(syspath, "ae");
    if (fptr == NULL ) goto err;

    char pid[20 + 1] = {0};
    snprintf(pid, 20, "%d", getpid());
    if (fputs(pid, fptr) < 0) {
        fclose(fptr);
        goto err;
    }

    fclose(fptr);

    return 0;
err:
    error_log("failed to move process into cgroup path %s: %s\n", syspath, strerror(errno));
    return -1;
}

int move_into_cgroup(pid_t pid)
{
    char path[PATH_MAX + 1] = {0};
    snprintf(path, PATH_MAX, "/proc/%d/cgroup", pid);
    FILE *fptr = fopen(path, "re");
    if (fptr == NULL) goto err;

    bool found = false;
    char entry[PATH_MAX + 1] = {0};
    while(fgets(entry, PATH_MAX, fptr) != NULL) {
        // We only support cgroup v2
        if ((strlen(entry) < 5) || (entry[0] != '0')) continue;

        if (do_move_into_cgroup(entry) < 0) break;
        found = true;
        break;
    }

    fclose(fptr);

    if (!found) {
        error_log("failed to move process into cgroup or cgroup v2 not found\n");
        return -1;
    }

    return 0;
err:
    error_log("failed to move process into cgroup: %s\n", strerror(errno));
    return -1;
}

int move_into_namespaces_compat(int pid)
{
    char path[PATH_MAX + 1] = {0};
    snprintf(path, PATH_MAX, "/proc/%d/ns", pid);
    int nsfd = open(path, O_RDONLY | O_CLOEXEC);
    if (nsfd == -1) goto err;

    // We must not join the user namespace so virtiofsd can keep its capabilities
    char *ns[6] = {"cgroup", "ipc", "mnt", "net", "pid", "uts"};
    for (int i = 0; i < 6; i++) {
        int fd = openat(nsfd, ns[i], O_RDONLY | O_CLOEXEC);
        if (fd == -1) goto err;
        if (setns(fd, 0) == -1) goto err;
        close(fd);
    }
    close(nsfd);

    return 0;
err:
    error_log("failed to move process into the namespace: %s\n", strerror(errno));
    return -1;
}

int move_into_namespaces(pid_t pid)
{
    fprintf(stderr, "move the process into same namespaces as %d\n", pid);
    int fd = syscall(SYS_pidfd_open, pid, 0);
    if (fd < 0) {
        if (errno == ENOSYS) {
            // pidfd_open() requires kernel 5.3 and above,
            // let's join each NS one by one on older kernels.
            return move_into_namespaces_compat(pid);
        }
        goto err;
    }

    // We must not join the user namespace so virtiofsd can keep its capabilities
    if (setns(fd, CLONE_NEWNET
                  | CLONE_NEWPID
                  | CLONE_NEWIPC
                  | CLONE_NEWNS
                  | CLONE_NEWCGROUP
                  | CLONE_NEWUTS) < 0) goto err;

    return 0;
err:
    error_log("failed to move process into the namespace: %s\n", strerror(errno));
    return -1;
}

int main(int argc, char **argv)
{
    struct arguments args = {0};
    if (parse_arguments(argc, argv, &args) < 0) {
        usage();
        exit(EXIT_FAILURE);
    }

    if(move_into_cgroup(args.pid) < 0 ) exit(EXIT_FAILURE);
    if(move_into_namespaces(args.pid) < 0 ) exit(EXIT_FAILURE);

    // Let's make sure we only run one instance of virtiofsd.
    //
    // The main idea is to lock a file and "leak" the file descriptor
    // into virtiofsd, since the lock is preserved across the execve()
    // call. It will be automatically released when the file descriptor
    // is closed at virtiofsd exit.
    //
    // We must do this here, after entering the mount namespace
    // but before re-parenting under the placeholder, otherwise
    // the placeholder will exit if we quit.
    int fd = open("/var/run/virtiofsd.lock", O_RDONLY | O_CREAT, S_IRUSR);
    if (fd < 0) {
        error_log("failed to open the lock file: %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    int ret = flock(fd, LOCK_EX | LOCK_NB);
    if (ret < 0) {
        if (errno == EWOULDBLOCK) {
            // virtiofsd is already running, we must not return an error here,
            // otherwise the dispatcher will be re-queued and executed again
            // and again endlessly.
            exit(EXIT_SUCCESS);
        }
        exit(EXIT_FAILURE);
    }

    // The PID namespace is special in the sense that a fork() is
    // required after calling setns() to actually enter the PID NS.
    //
    // Since we want to re-parent virtiofsd to be a child of the
    // PID 1 inside the container, we really need to fork() twice (see
    // daemon()), because when a child process becomes orphaned, it is
    // re-parented to the "init" process in the PID NS of its _parent_,
    // so make sure the virtiofsd's parent process is already inside the
    // PId NS.
    pid_t child =  fork();
    if (child < 0) exit(EXIT_FAILURE);
    if (child > 0) exit(EXIT_SUCCESS);

    if (daemon(0, -1) != 0) {
        error_log("failed daemon: %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    if (freopen("/proc/1/fd/1", "a", stdout) == NULL) {
        error_log("failed redirecting stdout: %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    if (freopen("/proc/1/fd/2", "a", stderr) == NULL){
        error_log("failed redirecting stdout: %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    fprintf(stderr, "start virtiofsd\n");

    // Let's run virtiofsd:
    // - chrooting it inside the shared dir, without
    // CAP_MKNOD to disable the creation of devices (besides FIFOs).
    // - use file handles, if the filesystem supports them
    // (i.e., --inode-file-handles=prefer).
    // - use file handles for migration, and report any error to the
    // target guest. We keep CAP_DAC_READ_SEARCH since is required in the
    // target to open the file handles.
    // - squash all UIDs/GIDs in the guest to the non-root UID defined in
    // 'util.NonRootUID' (i.e., 107). So, all files will be created with that UID/GID
    // even if virtiofsd runs as root.
    char bin[] = "/usr/libexec/virtiofsd";
    char *virtiofs_argv[] = {
        bin,
        "--socket-path", args.socket_flag,
        "--shared-dir", args.shareddir_flag,
        "--cache", "auto",
        "--sandbox", "chroot",
        "--modcaps=+dac_read_search:-mknod",
        "--inode-file-handles=prefer",
        "--migration-mode=file-handles",
        "--migration-on-error=guest-error",
        "--translate-uid=squash-guest:0:107:4294967295",
        "--translate-gid=squash-guest:0:107:4294967295",
        "--xattr", NULL };
    char *env[] = { NULL };

    if (execve(bin, virtiofs_argv, env) < 0) {
        error_log("failed executing virtiofsd: %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    exit(EXIT_SUCCESS);
}
