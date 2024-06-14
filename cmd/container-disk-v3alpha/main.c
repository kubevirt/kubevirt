#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <unistd.h>
#include <getopt.h>
#include <errno.h>
#include <signal.h>

#include <time.h>

char pidfile[] = "/var/run/containerdisk/pidfile";

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
	unlink(pidfile);
	printf("Gracefully terminating\n");
	exit(0);
}

int main(int argc, char **argv) {
	static struct option long_options[] = {
		{"no-op", 0, 0, 'n'},
		{0, 0, 0, 0}
	};
	int pid = getpid();
	FILE *file;
	int c;

	c = getopt_long(argc, argv, "c:", long_options, NULL);
	if (c == 'n')
		exit(0);
	if (signal(SIGTERM, sig_handler) == SIG_ERR) {
		error_log("failed to register SIGTERM callback\n");
		exit(1);
	}
	if (signal(SIGINT, sig_handler) == SIG_ERR) {
		error_log("failed to register SIGINT callback\n");
		exit(1);
	}

	if (!(file = fopen(pidfile, "w"))) {
		error_log("Failed to open pidfile\n");
		exit(1);
	}

	fprintf(file, "%d", pid);
	fclose(file);
	pause();
}
