/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

#define _GNU_SOURCE

#include <arpa/inet.h>
#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <linux/capability.h>
#include <netinet/in.h>
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <sys/prctl.h>
#include <sys/socket.h>
#include <sys/syscall.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <time.h>
#include <unistd.h>

#define DEFAULT_LAUNCHER "/usr/bin/virt-launcher"
#define DEFAULT_CONTAINER_DISK_DIR "/var/run/kubevirt/container-disks"
#define PASST_LOG_FILE "/var/run/kubevirt/passt.log"
#define QEMU_LOG_DIR "/run/kubevirt-private/libvirt/qemu/log"
#define ENVOY_READY_PORT 15021
#define ENVOY_QUIT_PORT 15020
#define HTTP_TIMEOUT_SEC 2
#define LOG_LINE_LIMIT (512 * 1024)

#define LOG(fmt, ...) fprintf(stderr, "virt-launcher-monitor: " fmt "\n", ##__VA_ARGS__)

static volatile sig_atomic_t got_chld;
static volatile sig_atomic_t shutdown_requested;
static volatile sig_atomic_t termination_sent;
static volatile sig_atomic_t launcher_exit_code = -1;
static pid_t launcher_pid;

static void handle_signal(int signo)
{
	if (signo == SIGCHLD) {
		got_chld = 1;
		return;
	}
	shutdown_requested = 1;
}

static int install_termination_handlers(void)
{
	struct sigaction action;

	memset(&action, 0, sizeof(action));
	action.sa_handler = handle_signal;
	sigemptyset(&action.sa_mask);

	if (sigaction(SIGINT, &action, NULL) < 0 ||
	    sigaction(SIGTERM, &action, NULL) < 0 ||
	    sigaction(SIGQUIT, &action, NULL) < 0) {
		return -1;
	}
	return 0;
}

static int install_sigchld_handler(void)
{
	struct sigaction action;

	memset(&action, 0, sizeof(action));
	action.sa_handler = handle_signal;
	sigemptyset(&action.sa_mask);
	action.sa_flags = SA_NOCLDSTOP;

	return sigaction(SIGCHLD, &action, NULL);
}

static int raise_net_bind_capability(void)
{
	struct __user_cap_header_struct header = {
		.version = _LINUX_CAPABILITY_VERSION_3,
		.pid = 0,
	};
	struct __user_cap_data_struct data[2];

	memset(data, 0, sizeof(data));
	if (syscall(SYS_capget, &header, data) < 0) {
		return -1;
	}

	data[0].effective |= (1u << CAP_NET_BIND_SERVICE);
	data[0].permitted |= (1u << CAP_NET_BIND_SERVICE);
	data[0].inheritable |= (1u << CAP_NET_BIND_SERVICE);

	if (syscall(SYS_capset, &header, data) < 0) {
		return -1;
	}
	if (prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, CAP_NET_BIND_SERVICE, 0, 0) < 0) {
		return -1;
	}
	return 0;
}

static bool is_keep_after_failure_arg(const char *arg)
{
	return strcmp(arg, "--keep-after-failure") == 0 ||
	       strncmp(arg, "--keep-after-failure=", 21) == 0;
}

static const char *container_disk_dir_from_args(int argc, char **argv)
{
	int i;

	for (i = 1; i < argc; i++) {
		if (strncmp(argv[i], "--container-disk-dir=", 21) == 0) {
			return argv[i] + 21;
		}
		if (strcmp(argv[i], "--container-disk-dir") == 0 && i + 1 < argc) {
			return argv[i + 1];
		}
	}
	return DEFAULT_CONTAINER_DISK_DIR;
}

static bool keep_after_failure_from_args(int argc, char **argv)
{
	bool keep = false;
	int i;

	for (i = 1; i < argc; i++) {
		if (strcmp(argv[i], "--keep-after-failure") == 0) {
			keep = true;
			continue;
		}
		if (strncmp(argv[i], "--keep-after-failure=", 21) == 0) {
			const char *value = argv[i] + 21;
			keep = strcmp(value, "false") != 0 && strcmp(value, "0") != 0;
		}
	}
	return keep;
}

static int filter_launcher_args(int argc, char **argv, char **out)
{
	int i, n = 0;

	out[n++] = argv[0];
	for (i = 1; i < argc; i++) {
		if (is_keep_after_failure_arg(argv[i])) {
			continue;
		}
		out[n++] = argv[i];
	}
	out[n] = NULL;
	return n;
}

static void sleep_ms(int ms)
{
	struct timespec ts = {
		.tv_sec = ms / 1000,
		.tv_nsec = (long)(ms % 1000) * 1000000L,
	};

	while (nanosleep(&ts, &ts) < 0 && errno == EINTR) {
	}
}

static void record_launcher_status(int status)
{
	if (WIFEXITED(status)) {
		launcher_exit_code = WEXITSTATUS(status);
	} else if (WIFSIGNALED(status)) {
		launcher_exit_code = 128 + WTERMSIG(status);
	} else {
		launcher_exit_code = 1;
	}
}

static void reap_children(void)
{
	int status;
	pid_t pid;

	while ((pid = waitpid(-1, &status, WNOHANG)) > 0) {
		if (pid == launcher_pid) {
			record_launcher_status(status);
		}
	}
}

static bool cmdline_contains(const char *cmdline, size_t len, const char *needle)
{
	size_t nlen = strlen(needle);

	if (nlen == 0 || len < nlen) {
		return false;
	}
	return memmem(cmdline, len, needle, nlen) != NULL;
}

static pid_t find_qemu_pid(const char *prefix)
{
	DIR *dir = opendir("/proc");
	struct dirent *entry;
	pid_t found = 0;

	if (dir == NULL) {
		return 0;
	}

	while ((entry = readdir(dir)) != NULL) {
		char path[64];
		char buf[4096];
		ssize_t n;
		int fd;
		char *end = NULL;
		long pid;

		if (entry->d_name[0] < '0' || entry->d_name[0] > '9') {
			continue;
		}
		pid = strtol(entry->d_name, &end, 10);
		if (end == NULL || *end != '\0' || pid <= 0) {
			continue;
		}

		snprintf(path, sizeof(path), "/proc/%ld/cmdline", pid);
		fd = open(path, O_RDONLY | O_CLOEXEC);
		if (fd < 0) {
			continue;
		}
		n = read(fd, buf, sizeof(buf));
		close(fd);
		if (n <= 0) {
			continue;
		}
		if (cmdline_contains(buf, (size_t)n, prefix)) {
			found = (pid_t)pid;
			break;
		}
	}

	closedir(dir);
	return found;
}

static bool pid_exists(pid_t pid)
{
	return kill(pid, 0) == 0 || errno == EPERM;
}

static int cleanup_qemu(void)
{
	const char *prefix = "qemu-system";
	pid_t pid = find_qemu_pid(prefix);
	int i;

	if (pid <= 0) {
		prefix = "qemu-kvm";
		pid = find_qemu_pid(prefix);
	}
	if (pid <= 0) {
		return 0;
	}

	LOG("Killing QEMU gracefully.");
	if (kill(pid, SIGTERM) < 0) {
		LOG("failed to signal QEMU: %s", strerror(errno));
		return -1;
	}

	for (i = 0; i < 100; i++) {
		if (!pid_exists(pid) && find_qemu_pid(prefix) <= 0) {
			return 0;
		}
		sleep_ms(100);
	}

	LOG("QEMU did not exit within 10 seconds");
	return -1;
}

static void dump_log_file(const char *path)
{
	FILE *file = fopen(path, "r");
	char *line = NULL;
	size_t cap = 0;
	ssize_t n;

	if (file == NULL) {
		if (errno != ENOENT) {
			LOG("failed to open file %s: %s", path, strerror(errno));
		}
		return;
	}

	LOG("dump log file: %s", path);
	while ((n = getline(&line, &cap, file)) >= 0) {
		if (n > LOG_LINE_LIMIT) {
			line[LOG_LINE_LIMIT] = '\0';
			LOG("%s: %s [line truncated]", path, line);
		} else if (n > 0 && line[n - 1] == '\n') {
			line[n - 1] = '\0';
			LOG("%s: %s", path, line);
		} else {
			LOG("%s: %s", path, line);
		}
	}
	free(line);
	fclose(file);
}

static void dump_launcher_logs(void)
{
	DIR *dir;
	struct dirent *entry;

	dump_log_file(PASST_LOG_FILE);

	dir = opendir(QEMU_LOG_DIR);
	if (dir == NULL) {
		if (errno != ENOENT) {
			LOG("failed to read qemu log directory: %s", strerror(errno));
		}
		return;
	}
	while ((entry = readdir(dir)) != NULL) {
		char path[512];

		if (entry->d_name[0] == '.') {
			continue;
		}
		snprintf(path, sizeof(path), "%s/%s", QEMU_LOG_DIR, entry->d_name);
		dump_log_file(path);
	}
	closedir(dir);
}

static void cleanup_container_disks(const char *directory)
{
	DIR *dir = opendir(directory);
	struct dirent *entry;

	if (dir == NULL) {
		return;
	}
	while ((entry = readdir(dir)) != NULL) {
		size_t len = strlen(entry->d_name);
		char path[512];

		if (len < 5 || strcmp(entry->d_name + len - 5, ".sock") != 0) {
			continue;
		}
		snprintf(path, sizeof(path), "%s/%s", directory, entry->d_name);
		if (unlink(path) < 0 && errno != ENOENT) {
			LOG("failed to remove %s: %s", path, strerror(errno));
		}
	}
	closedir(dir);
}

static int retry_delay_ms(int attempt)
{
	int delay = 10;
	int i;

	for (i = 0; i <= attempt; i++) {
		delay *= 5;
	}
	return delay;
}

static int http_request(int port, const char *request, int *status, char *server, size_t server_len)
{
	int fd;
	struct sockaddr_in addr;
	struct timeval timeout = { .tv_sec = HTTP_TIMEOUT_SEC, .tv_usec = 0 };
	char buf[2048];
	size_t off = 0;
	ssize_t n;
	char *line, *saveptr, *hdr;
	int got_status = 0;

	*status = 0;
	if (server_len > 0) {
		server[0] = '\0';
	}

	fd = socket(AF_INET, SOCK_STREAM | SOCK_CLOEXEC, 0);
	if (fd < 0) {
		return -errno;
	}
	if (setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout)) < 0 ||
	    setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout)) < 0) {
		int err = -errno;
		close(fd);
		return err;
	}

	memset(&addr, 0, sizeof(addr));
	addr.sin_family = AF_INET;
	addr.sin_port = htons((uint16_t)port);
	addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
	if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
		int err = -errno;
		close(fd);
		return err;
	}

	if (write(fd, request, strlen(request)) < 0) {
		int err = -errno;
		close(fd);
		return err;
	}

	while (off < sizeof(buf) - 1) {
		n = read(fd, buf + off, sizeof(buf) - 1 - off);
		if (n < 0) {
			int err = -errno;
			close(fd);
			return err;
		}
		if (n == 0) {
			break;
		}
		off += (size_t)n;
		buf[off] = '\0';
		if (strstr(buf, "\r\n\r\n") != NULL || strstr(buf, "\n\n") != NULL) {
			break;
		}
	}
	close(fd);
	buf[off] = '\0';

	line = strtok_r(buf, "\r\n", &saveptr);
	if (line == NULL || sscanf(line, "HTTP/%*s %d", status) != 1) {
		return -EPROTO;
	}
	got_status = 1;
	while ((hdr = strtok_r(NULL, "\r\n", &saveptr)) != NULL) {
		if (strncasecmp(hdr, "Server:", 7) == 0) {
			const char *value = hdr + 7;

			while (*value == ' ' || *value == '\t') {
				value++;
			}
			if (server_len > 0) {
				snprintf(server, server_len, "%s", value);
			}
		}
	}
	return got_status ? 0 : -EPROTO;
}

static bool is_retryable(int err)
{
	err = err < 0 ? -err : err;
	return err == ECONNREFUSED || err == ECONNRESET || err == ETIMEDOUT || err == EAGAIN;
}

static bool istio_proxy_present(void)
{
	const char *req = "GET /healthz/ready HTTP/1.0\r\nHost: localhost\r\n\r\n";
	int attempt;

	for (attempt = 0; attempt < 4; attempt++) {
		int status = 0;
		char server[128];
		int err = http_request(ENVOY_READY_PORT, req, &status, server, sizeof(server));

		if (err == 0 && strcasecmp(server, "envoy") == 0) {
			return true;
		}
		if (err < 0 && attempt < 3 && is_retryable(err)) {
			sleep_ms(retry_delay_ms(attempt));
			continue;
		}
		break;
	}
	return false;
}

static void terminate_istio_proxy(void)
{
	const char *req = "POST /quitquitquit HTTP/1.0\r\nHost: localhost\r\nContent-Length: 0\r\n\r\n";
	int attempt;

	if (!istio_proxy_present()) {
		return;
	}

	for (attempt = 0; attempt < 4; attempt++) {
		int status = 0;
		char server[8];
		int err = http_request(ENVOY_QUIT_PORT, req, &status, server, sizeof(server));

		if (err == 0 && status == 200) {
			return;
		}
		if ((err == 0 && status == 503) || (err < 0 && is_retryable(err))) {
			if (attempt < 3) {
				sleep_ms(retry_delay_ms(attempt));
				continue;
			}
		}
		if (err == 0) {
			LOG("Istio quit request returned HTTP %d", status);
			return;
		}
		LOG("Istio quit request failed: %s", strerror(err < 0 ? -err : err));
		return;
	}
	LOG("all attempts to terminate istio-proxy failed");
}

static int run_launcher(int argc, char **argv)
{
	char *child_argv[argc + 1];
	const char *launcher = getenv("VIRT_LAUNCHER");
	sigset_t block_chld, old_mask;
	pid_t pid;

	if (launcher == NULL || launcher[0] == '\0') {
		launcher = DEFAULT_LAUNCHER;
	}

	filter_launcher_args(argc, argv, child_argv);
	child_argv[0] = (char *)launcher;

	sigemptyset(&block_chld);
	sigaddset(&block_chld, SIGCHLD);
	if (sigprocmask(SIG_BLOCK, &block_chld, &old_mask) < 0) {
		LOG("failed to block SIGCHLD: %s", strerror(errno));
		return 1;
	}

	pid = fork();
	if (pid < 0) {
		int err = errno;
		sigprocmask(SIG_SETMASK, &old_mask, NULL);
		LOG("failed to run %s: %s", launcher, strerror(err));
		return 1;
	}
	if (pid == 0) {
		sigprocmask(SIG_SETMASK, &old_mask, NULL);
		/*
		 * Tests set VIRT_LAUNCHER to a fake binary that has no
		 * file capabilities. Production always execs the default
		 * path and must fail closed if ambient caps cannot be set.
		 */
		if (getenv("VIRT_LAUNCHER") == NULL && raise_net_bind_capability() < 0) {
			LOG("failed to raise CAP_NET_BIND_SERVICE: %s", strerror(errno));
			_exit(1);
		}
		execv(launcher, child_argv);
		LOG("failed to exec %s: %s", launcher, strerror(errno));
		_exit(127);
	}

	launcher_pid = pid;
	if (install_sigchld_handler() < 0) {
		sigprocmask(SIG_SETMASK, &old_mask, NULL);
		LOG("failed to install SIGCHLD handler: %s", strerror(errno));
		return 1;
	}
	if (sigprocmask(SIG_SETMASK, &old_mask, NULL) < 0) {
		LOG("failed to unblock SIGCHLD: %s", strerror(errno));
		return 1;
	}

	while (launcher_exit_code < 0) {
		if (shutdown_requested && !termination_sent) {
			LOG("signalling virt-launcher to shut down");
			if (kill(launcher_pid, SIGTERM) < 0) {
				LOG("received signal but can't signal virt-launcher to shut down: %s",
				    strerror(errno));
			}
			termination_sent = 1;
		}
		if (got_chld) {
			got_chld = 0;
			reap_children();
			continue;
		}
		sleep_ms(10);
		reap_children();
	}
	return launcher_exit_code;
}

int main(int argc, char **argv)
{
	const char *disk_dir = container_disk_dir_from_args(argc, argv);
	bool keep = keep_after_failure_from_args(argc, argv);
	int exit_code;
	int monitor_error = 0;

	if (install_termination_handlers() < 0) {
		LOG("failed to install signal handlers: %s", strerror(errno));
		return 1;
	}

	exit_code = run_launcher(argc, argv);
	if (exit_code != 0) {
		LOG("dirty virt-launcher shutdown: exit-code %d", exit_code);
	}

	dump_launcher_logs();
	if (cleanup_qemu() < 0) {
		monitor_error = 1;
	}
	cleanup_container_disks(disk_dir);
	terminate_istio_proxy();

	if (keep && (monitor_error || exit_code != 0)) {
		LOG("keeping virt-launcher container alive since --keep-after-failure is set to true");
		for (;;) {
			pause();
		}
	}

	if (monitor_error) {
		return 1;
	}
	LOG("Exiting...");
	return exit_code;
}
