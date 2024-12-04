// SPDX-License-Identifier: Apache-2.0

#define _GNU_SOURCE
#include <getopt.h>
#include <stdio.h>
#include <stdarg.h>
#include <string.h>
#include <stdlib.h>
#include <limits.h>
#include <unistd.h>
#include <time.h>
#include <errno.h>
#include <sched.h>
#include <sys/syscall.h>
#include <fcntl.h>
#include <sys/file.h>

struct arguments {
    char socket_flag[PATH_MAX];
    char shareddir_flag[PATH_MAX];
    int pid;
};

struct arguments args;

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
    exit(EXIT_FAILURE);
}

#define FMT_SZ 50
void error_log(const char *format, ...)
{
    va_list arglist;

    time_t ltime; /* calendar time */
    ltime=time(NULL); /* get current cal time */

    char time_fmt[FMT_SZ] = {};
    strftime(time_fmt, FMT_SZ, "%b %d %H:%M:%S ", localtime(&ltime));
    fprintf(stderr, "%s", time_fmt);

    fprintf(stderr, "error: ");
    va_start(arglist, format);
    vfprintf(stderr, format, arglist);
    va_end(arglist);
}

void parse_arguments(int argc, char **argv, struct arguments *args)
{
    int c;
    while(1) {
        int option_index = 0;
        int c = getopt_long(argc, argv, "d:p:s:", long_options, &option_index);

        if (c == -1) break;

        switch (c) {
            case 'd':
                strncpy(args->shareddir_flag, optarg, strlen(optarg));
                break;
            case 's':
                strncpy(args->socket_flag, optarg, strlen(optarg));
                break;
            case 'p':
                args-> pid = atoi(optarg);
                break;
            case '?': /* fallthrough */
            default:
                usage();
                break;
        }
    }

    if (args->pid < 1) {
        error_log("pid needs to be set");
        usage();
        exit(EXIT_FAILURE);
    }
}

int move_into_cgroup(pid_t pid)
{
    char path[PATH_MAX - 30];
    char syspath[PATH_MAX];
    FILE *fptr;
    char str[20];

    snprintf(path, PATH_MAX, "/proc/%d/cgroup", pid);
    fptr = fopen(path, "r");
    if (fptr == NULL) goto err;
    fgets(path, sizeof(path), fptr);
    fclose(fptr);

    snprintf(path, strlen(path) - 4, path + 4);
    if (strcmp(path, "") == 0) {
        snprintf(syspath, PATH_MAX, "/sys/fs/cgroup/cgroup.procs");
    } else {
        snprintf(syspath, PATH_MAX, "/sys/fs/cgroup/%s/cgroup.procs", path);
    }

    fprintf(stderr, "move the process into the cgroup as %s\n", syspath);
    fptr = fopen(syspath, "a");
    if (fptr == NULL ) goto err;
    sprintf(str, "%d", getpid());
    fputs(str, fptr);
    fclose(fptr);

    return 0;
err:
    error_log("failed to move process into cgroup path %s: %s", syspath, strerror(errno));
    return -1;
}

int move_into_namespaces(pid_t pid)
{
    fprintf(stderr, "move the process into same namespaces as %d\n", pid);
    int fd = syscall(SYS_pidfd_open, pid, 0);
    if (fd < 0) goto err;

    if (setns(fd, CLONE_NEWNET
                  | CLONE_NEWPID
                  | CLONE_NEWIPC
                  | CLONE_NEWNS
                  | CLONE_NEWCGROUP
                  | CLONE_NEWUTS) < 0) goto err;

    return 0;
err:
    error_log("failed to move process into the namespace: %s", strerror(errno));
    return -1;
}

int main(int argc, char **argv)
{
    parse_arguments(argc, argv, &args);

    if(move_into_cgroup(args.pid) < 0 ) exit(EXIT_FAILURE);
    if(move_into_namespaces(args.pid) < 0 ) exit(EXIT_FAILURE);

    /*
     Let's makes sure we only run one instance of virtiofsd.

     The main idea is to lock a file and "leak" the file descriptor
     into virtiofsd, since the lock is preserved across the execve()
     call. It will be automatically released when the file descriptor
     is closed at virtiofsd exit.

     We must do this here, after entering the mount namespace
     but before re-parenting under the placeholder, otherwise
     the placeholder will exit if we quit.
     */
    int fd = open("/var/run/virtiofsd.lock", O_RDONLY | O_CREAT, S_IRUSR);
    if (fd < 0) {
        error_log("failed to open the lock file: %s", strerror(errno));
        exit(EXIT_FAILURE);
    }

    int ret = flock(fd, LOCK_EX | LOCK_NB);
    if (ret < 0) {
        if (errno == EWOULDBLOCK) {
            /*
             virtiofsd is already running, we must not return an error here,
             otherwise the dispatcher will be re-queued and executed again
             and again endlessly.
             */
            exit(EXIT_SUCCESS);
        }
        exit(EXIT_FAILURE);
    }

    /*
     The PID namespace is special in the sense that a fork() is
     required after calling setns() to actually enter the PID NS.

     Since we want to re-parent virtiofsd to be a child of the
     PID 1 inside the container, we really need to fork() twice (see
     daemon()), because when a child process becomes orphaned, it is
     re-parented to the "init" process in the PID NS of its _parent_,
     so make sure the virtiofsd's parent process is already inside the
     PId NS.
     */
    pid_t child =  fork();
    if (child < 0) exit(EXIT_FAILURE);
    if (child > 0) exit(EXIT_SUCCESS);

    if (daemon(0, -1) != 0) {
        error_log("failed daemon: %s", strerror(errno));
        exit(EXIT_FAILURE);
    }

    if (freopen("/proc/1/fd/1", "a", stdout) == NULL) {
        error_log("failed redirecting stdout: %s", strerror(errno));
        exit(EXIT_FAILURE);
    }

    if (freopen("/proc/1/fd/2", "a", stderr) == NULL){
        error_log("failed redirecting stdout: %s", strerror(errno));
        exit(EXIT_FAILURE);
    }

    fprintf(stderr, "start virtiofsd\n");

    /*
     Let's run virtiofsd:
     - chrooting it inside the shared dir, without
     CAP_MKNOD to disable the creation of devices (besides FIFOs).
     - use file handles, if the filesystem supports them
     (i.e., --inode-file-handles=prefer).
     - use file handles for migration, and report any error to the
     target guest. We keep CAP_DAC_READ_SEARCH since is required in the
     target to open the file handles.
     - squash all UIDs/GIDs in the guest to the non-root UID defined in
     'util.NonRootUID' (i.e., 107). So, all files will be created with that UID/GID
     even if virtiofsd runs as root.
    */
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
        error_log("failed executing virtiofsd: %s", strerror(errno));
        exit(EXIT_FAILURE);
    }
}
