#include <errno.h>
#include <getopt.h>
#include <libgen.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/un.h>
#include <unistd.h>
#include <signal.h>
#include <fcntl.h>

#include <time.h>

#define LISTEN_BACKLOG 50

char copy_path[108];

void error_log(const char *format, ...)
{
    va_list arglist;

    time_t ltime; /* calendar time */
    ltime=time(NULL); /* get current cal time */
    fprintf(stderr, "%s",asctime(localtime(&ltime)));
    fprintf(stderr, "error: ");
    va_start(arglist, format);
    vfprintf(stderr, format, arglist);
    va_end(arglist);
}

void sig_handler(int signo) {
    if (copy_path != NULL) {
        unlink(copy_path);
    }
    exit(0);
}

static void *socket_check(int fd, void *arg) {
    struct stat st = {0};
    char *copy_path;
    copy_path = (char *)arg;
    int connfd;
    bool connReceived = false;

    /*
     * Periodically check the following:
     *
     * First, if the socket file still exists. We use it as an indicator to
     * shut down the container, in case that we don't receive a signal from
     * kuberenetes. We had issues with receiving the signal in time on
     * different container runtime implementations over time and therefore use
     * this as a precaution.
     *
     * Second accept socket connections to avoid filling up the SYN queue. If
     * ther is a connection, we close it immediatley and immediately try to
     * read the next connection until there are no more connections in the
     * queue. Once the queue is empty, we fall back to sleep for a second.
     *
     * If within that second more than 50 connections enter the queue, then we
     * clearly have a bug in virt-handler.
     */
    for (;;) {
        if (!connReceived) {
            sleep(1);
        }
        if (stat(copy_path, &st) == -1) {
            error_log("socket %s does not exist anymore\n", copy_path);
            exit(0);
        }
        connfd = accept(fd, (struct sockaddr*)NULL, NULL);
        if (connfd == -1) {
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                /* nothing to do */
                connReceived = false;
                continue;
            } else {
                error_log("failed to accept connections on the socket: %d\n", errno);
                exit(1);
            }
        }
        connReceived = true;
        /* connection received, the only thing to do is closing the connection */
        close(connfd);
    }
}

int main(int argc, char **argv) {
    char *copy_path_dir;
    char *copy_path_tmp;

    if (signal(SIGTERM, sig_handler) == SIG_ERR) {
        error_log("failed to register SIGTERM callback\n");
        exit(1);
    }

    int c;
    while(1) {
        static struct option long_options[] = {
            /* These options don’t set a flag.
                We distinguish them by their indices. */
            {"copy-path",    required_argument, 0, 'c'},
            {"no-op", 0, 0, 'n'},
            {0, 0, 0, 0}
        };
        /* getopt_long stores the option index here. */
        int option_index = 0;
        c = getopt_long(argc, argv, "c:", long_options, &option_index);

        /* Detect the end of the options. */
        if (c == -1) {
            break;
        }

        switch (c) {
            case 'c':
                // copy_path limited by size of address.sun_path with .sock suffix
                // copy_path + .sock + terminating null byte = 108 chars
                // 102       + 5     + 1                     = 108
                if (strlen(optarg) > 102) {
                    error_log("copy path can not be longer than 102 characters\n");
                    exit(1);
                }
                strncpy(copy_path, optarg, strlen(optarg));
                copy_path_tmp = strndup(copy_path, strlen(copy_path));
                copy_path_dir = dirname(copy_path_tmp);
                break;
            case 'n':
		exit(0);
            case '?':
                exit(1);
            default:
                abort();
        }
    }

    struct stat st = {0};

    if (stat(copy_path_dir, &st) == -1) {
        if (mkdir(copy_path_dir, 0777) != 0) {
            error_log("failed to create disk directory %s\n", copy_path_dir);
            exit(1);
        }
    }
    free(copy_path_tmp);

    struct sockaddr_un address;
    /*
    * For portability clear the whole structure, since some
    * implementations have additional (nonstandard) fields in
    * the structure.
    */
    memset(&address, 0, sizeof(struct sockaddr_un));
    address.sun_family = AF_UNIX;
    strcat(copy_path, ".sock");
    strncpy(address.sun_path, copy_path, sizeof(address.sun_path));

    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd == -1) {
        error_log("failed to create socket on %s\n", copy_path);
        exit(1);
    }

    /* make the socket non-blocking */
    fcntl(fd, F_SETFL, O_NONBLOCK);

    if (bind(fd, (struct sockaddr*)(&address), sizeof(struct sockaddr_un)) == -1) {
        error_log("failed to bind socket %s\n", copy_path);
        exit(1);
    }

    if (listen(fd, LISTEN_BACKLOG) == -1) {
        error_log("failed to listen socket %s\n", copy_path);
        exit(1);
    }

    socket_check(fd, (void *)copy_path);
}
