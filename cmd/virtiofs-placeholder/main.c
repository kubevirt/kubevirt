// SPDX-License-Identifier: Apache-2.0
/*
 * virtiofsd placeholder
 *
 * The purpose of this command is to function as PID 1 inside the container having
 * the same lifetime as virtiofsd.
 *
 * The dispatcher will get the PID of this command by connecting to the socket,
 * and will run a privileged virtiofsd on the same namespaces and cgroup as this command.
 *
 * Since virtiofsd will be re-parented as a child of this command, it should terminate
 * when it receives the SIGCHLD signal indicating that virtiofsd is finished.
 */

#include <errno.h>
#include <getopt.h>
#include <limits.h>
#include <signal.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/epoll.h>
#include <sys/signalfd.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <time.h>
#include <unistd.h>

struct arguments {
    char socket[PATH_MAX + 1];
};

struct option long_options[] = {
    {"socket-path", required_argument, 0, 's'},
    {0, 0, 0, 0}
};

void usage()
{
    printf("Placeholder for virtiofs\n"
           "Usage:\n"
           "\t-s, --socket:\tContainer socket path to retrieve the pid\n");
}

int parse_arguments(int argc, char **argv, struct arguments *args)
{
    while(1) {
        int c = getopt_long(argc, argv, "s:", long_options, NULL);
        if (c == -1) break;

        switch (c) {
            case 's':
                strncpy(args->socket, optarg, PATH_MAX);
                break;
            case '?': // fallthrough
            default:
                return -1;
        }
    }
    return 0;
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

int get_signalfd(int signal)
{
    sigset_t sigset;
    sigemptyset(&sigset);
    sigaddset(&sigset, signal);
    if (sigprocmask(SIG_BLOCK, &sigset, NULL) == -1) {
        error_log("sigprocmask failed: %s\n", strerror(errno));
        return -1;
    }

    int fd = signalfd(-1, &sigset, SFD_NONBLOCK);
    if (fd < 0) {
        error_log("signalfd failed: %s\n", strerror(errno));
    }

    return fd;
}

int create_socket(const char *path)
{
    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd < 0) goto err;

    struct sockaddr_un addr = {0};
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, path, sizeof(addr.sun_path) - 1);

    if (bind(fd, (struct sockaddr *) &addr, sizeof(addr)) < 0) goto err_close;

    if (listen(fd, 1) < 0) goto err_close;

    return fd;
err_close:
    close(fd);
err:
    error_log("create_socket failed: %s\n", strerror(errno));
    return -1;
}

int monitor(int socket_fd, int sig_fd)
{
    int efd = epoll_create1(0);
    if (efd < 0) goto err;

    // Watch the socket
    // Even if we expect just one connection, we cannot use EPOLLONESHOT, because the dispatcher
    // could have died after connect() but before spawning virtiofsd, so we need to allow successive
    // connections.
    struct epoll_event socket_event = {.events = EPOLLIN, .data.fd = socket_fd};
    if (epoll_ctl(efd, EPOLL_CTL_ADD, socket_fd, &socket_event) < 0) goto err;

    struct epoll_event signal_event = {.events = EPOLLIN, .data.fd = sig_fd};
    if (epoll_ctl(efd, EPOLL_CTL_ADD, sig_fd, &signal_event) < 0) goto err;

    struct epoll_event epoll_events;
    while (true) {
        int ret = epoll_wait(efd, &epoll_events, 1, -1);
        if (ret < 0 && errno == EINTR) continue;
        if (ret < 0) goto err;

        if (epoll_events.data.fd == sig_fd) {
            // We received a SIGCHLD if virtiofsd exited, we must exit too
            struct signalfd_siginfo sfdi;
            int len = read(epoll_events.data.fd, &sfdi, sizeof(sfdi));
            // Let's assume that only virtiofsd will run with privileges (i.e., uid == 0)
            if (len == sizeof(sfdi) && sfdi.ssi_uid == 0) break;
        } else if (epoll_events.data.fd == socket_fd) {
            int accept_fd = accept(socket_fd, NULL, NULL);
            if (accept_fd < 0) goto err;

            // Get a notification if the socket is closed, to avoid leaking the FD
            struct epoll_event acceptfd_event = {.events = EPOLLRDHUP | EPOLLONESHOT, .data.fd = accept_fd};
            // Ignore the error, If epoll_ctl fails we will just leak the accept_fd
            if (epoll_ctl(efd, EPOLL_CTL_ADD, accept_fd, &acceptfd_event) < 0) {
                error_log("monitor failed to add accepted connection: %s\n", strerror(errno));
            }
        } else if (epoll_events.events & EPOLLRDHUP) {
            // An event from the accepted connection, the other side closed the connection
            close(epoll_events.data.fd);
        }
    }

    return 0;
err:
    error_log("monitor failed: %s\n", strerror(errno));
    return -1;
}

int main(int argc, char **argv)
{
    fprintf(stderr, "start monitoring for virtiofs\n");

    struct arguments args = {0};
    if(parse_arguments(argc, argv, &args) < 0) {
        usage();
        exit(EXIT_FAILURE);
    }

    // sig_fd and socket_fd will close on exit
    int sig_fd = get_signalfd(SIGCHLD);
    if (sig_fd == -1) exit(EXIT_FAILURE);

    int socket_fd = create_socket(args.socket);
    if (socket_fd == -1) exit(EXIT_FAILURE);

    if (monitor(socket_fd, sig_fd) == -1) exit(EXIT_FAILURE);

    exit(EXIT_SUCCESS);
}
