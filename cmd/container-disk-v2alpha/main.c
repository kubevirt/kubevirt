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
#include <time.h>

#define LISTEN_BACKLOG 50
#define READINESS_PROBE_FILE "/healthy"

char copy_path[108];

void error_log(const char *format, ...)
{
    va_list arglist;

    time_t ltime; /* calendar time */
    ltime=time(NULL); /* get current cal time */
    fprintf(stderr, "%s",asctime(localtime(&ltime)));
    fprintf(stderr, "error: ");
    va_start(arglist, format);
    fprintf(stderr, format, arglist);
    va_end(arglist);
}

void sig_handler(int signo) {
    if (copy_path != NULL) {
        unlink(copy_path);
        free(copy_path);
    }
    exit(0);
}

int main(int argc, char **argv) {
    char *copy_path_dir;
    char *copy_path_tmp;
    bool health_check = false;

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
            {"health-check", no_argument,       0, 'p'},
            {0, 0, 0, 0}
        };
        /* getopt_long stores the option index here. */
        int option_index = 0;
        c = getopt_long(argc, argv, "c:p", long_options, &option_index);

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

            case 'p':
                health_check = true;
                break;

            case '?':
                exit(1);
            default:
                abort();
        }
    }

    struct stat st = {0};
    if (health_check) {
        if (stat(READINESS_PROBE_FILE, &st) == -1) {
            error_log("readiness probe %s does not exist, errno: %d\n", READINESS_PROBE_FILE, errno);
            exit(1);
        } else {
            exit(0);
        }
    }

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
    strncat(copy_path, ".sock", 5);
    strncpy(address.sun_path, copy_path, sizeof(address.sun_path));

    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd == -1) {
        error_log("failed to create socket on %s\n", copy_path);
        exit(1);
    }

    if (bind(fd, (struct sockaddr*)(&address), sizeof(struct sockaddr_un)) == -1) {
        error_log("failed to bind socket %s\n", copy_path);
        exit(1);
    }

    if (listen(fd, LISTEN_BACKLOG) == -1) {
        error_log("failed to listen socket %s\n", copy_path);
        exit(1);
    }

    // Create readiness probe
    FILE *probe;
    probe = fopen(READINESS_PROBE_FILE, "w");
    if (probe == NULL) {
        error_log("failed to create readiness probe\n");
        exit(1);
    }
    fclose(probe);

    for (;;) {
        sleep(1);
        if (stat(copy_path, &st) == -1) {
            error_log("socket %s does not exist anymore, errno %d\n", copy_path, errno);
            exit(0);
        }
    }
}
