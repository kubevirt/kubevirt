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
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <sys/prctl.h>
#include <sys/socket.h>
#include <sys/syscall.h>
#include <sys/time.h>
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
#define LOG_ERROR(fmt, ...) fprintf(stderr, "virt-launcher-monitor: error: " fmt "\n", ##__VA_ARGS__)

static volatile sig_atomic_t termination_requests;
static volatile sig_atomic_t launcher_exit_code = -1;
static volatile sig_atomic_t launcher_pid;
static volatile sig_atomic_t wait_error;

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

/* waitpid is async-signal-safe. Reap every adopted child for the full lifetime
 * of PID 1, including children which become orphaned during cleanup. */
static void reap_children(void)
{
	int saved_errno = errno;
	int status;
	pid_t pid;

	for (;;) {
		pid = waitpid(-1, &status, WNOHANG);
		if (pid > 0) {
			if (pid == launcher_pid) {
				record_launcher_status(status);
			}
			continue;
		}
		if (pid < 0 && errno == EINTR) {
			continue;
		}
		if (pid < 0 && (errno != ECHILD || launcher_exit_code < 0)) {
			wait_error = 1;
		}
		break;
	}
	errno = saved_errno;
}

static void handle_signal(int signo)
{
	if (signo == SIGCHLD) {
		reap_children();
		return;
	}
	/* Match Go's buffered signal channel without overflowing sig_atomic_t. */
	if (termination_requests < 10) {
		termination_requests++;
	}
}

static int install_termination_handlers(void)
{
	struct sigaction action;

	memset(&action, 0, sizeof(action));
	action.sa_handler = handle_signal;
	sigemptyset(&action.sa_mask);
	sigaddset(&action.sa_mask, SIGINT);
	sigaddset(&action.sa_mask, SIGTERM);
	sigaddset(&action.sa_mask, SIGQUIT);

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

#ifndef VIRT_LAUNCHER_MONITOR_TESTING
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
#endif

static bool is_keep_after_failure_arg(const char *arg)
{
	return strcmp(arg, "--keep-after-failure") == 0;
}

static const char *container_disk_dir_from_args(int argc, char **argv)
{
	const char *directory = DEFAULT_CONTAINER_DISK_DIR;
	int i;

	for (i = 1; i < argc; i++) {
		if (strcmp(argv[i], "--") == 0) {
			break;
		}
		if (strncmp(argv[i], "--container-disk-dir=", 21) == 0) {
			directory = argv[i] + 21;
			continue;
		}
		if (strcmp(argv[i], "--container-disk-dir") == 0 && i + 1 < argc &&
		    strcmp(argv[i + 1], "--") != 0) {
			directory = argv[++i];
		}
	}
	return directory;
}

static bool parse_bool(const char *value, bool *parsed)
{
	if (strcmp(value, "1") == 0 || strcmp(value, "t") == 0 ||
	    strcmp(value, "T") == 0 || strcmp(value, "TRUE") == 0 ||
	    strcmp(value, "true") == 0 || strcmp(value, "True") == 0) {
		*parsed = true;
		return true;
	}
	if (strcmp(value, "0") == 0 || strcmp(value, "f") == 0 ||
	    strcmp(value, "F") == 0 || strcmp(value, "FALSE") == 0 ||
	    strcmp(value, "false") == 0 || strcmp(value, "False") == 0) {
		*parsed = false;
		return true;
	}
	return false;
}

static bool keep_after_failure_from_args(int argc, char **argv)
{
	bool keep = false;
	int i;

	for (i = 1; i < argc; i++) {
		if (strcmp(argv[i], "--") == 0) {
			break;
		}
		if (strcmp(argv[i], "--keep-after-failure") == 0) {
			keep = true;
		} else if (strncmp(argv[i], "--keep-after-failure=", 21) == 0) {
			bool parsed;
			if (parse_bool(argv[i] + 21, &parsed)) {
				keep = parsed;
			}
		}
	}
	return keep;
}

static int filter_launcher_args(int argc, char **argv, char **out)
{
	int i, n = 0;

	/* Match Go's removeArg: only the bare flag is removed. */
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
	if (kill(pid, SIGTERM) < 0 && errno != ESRCH) {
		LOG_ERROR("failed to signal QEMU: %s", strerror(errno));
		return -1;
	}

	for (i = 0; i < 100; i++) {
		if (!pid_exists(pid) && find_qemu_pid(prefix) <= 0) {
			return 0;
		}
		sleep_ms(100);
	}

	LOG_ERROR("QEMU did not exit within 10 seconds");
	return -1;
}

static void emit_log_line(const char *path, const char *line, size_t length, bool truncated)
{
	if (length > 0 && line[length - 1] == '\r') {
		length--;
	}
	fputs("virt-launcher-monitor: ", stderr);
	fwrite(path, 1, strlen(path), stderr);
	fputs(": ", stderr);
	fwrite(line, 1, length, stderr);
	if (truncated) {
		fputs(" [line truncated]", stderr);
	}
	fputc('\n', stderr);
}

/* Stream logs with fixed bounded storage; getline may allocate an unbounded
 * buffer before a long line can be truncated. */
static void dump_log_file(const char *path)
{
	FILE *file = fopen(path, "r");
	char *line;
	size_t length = 0;
	bool truncated = false;
	int ch;

	if (file == NULL) {
		if (errno != ENOENT) {
			LOG_ERROR("failed to open file %s: %s", path, strerror(errno));
		}
		return;
	}
	line = malloc(LOG_LINE_LIMIT);
	if (line == NULL) {
		LOG_ERROR("failed to allocate log line buffer for %s", path);
		fclose(file);
		return;
	}

	LOG("dump log file: %s", path);
	while ((ch = fgetc(file)) != EOF) {
		if (ch == '\n') {
			emit_log_line(path, line, length, truncated);
			length = 0;
			truncated = false;
			continue;
		}
		if (length < LOG_LINE_LIMIT) {
			line[length++] = (char)ch;
		} else {
			truncated = true;
		}
	}
	if (length > 0 || truncated) {
		emit_log_line(path, line, length, truncated);
	}
	free(line);
	if (ferror(file)) {
		int read_error = errno == 0 ? EIO : errno;
		LOG_ERROR("failed to read file %s: %s", path, strerror(read_error));
	}
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
			LOG_ERROR("failed to read qemu log directory: %s", strerror(errno));
		}
		return;
	}
	for (;;) {
		char path[512];
		int n;

		errno = 0;
		entry = readdir(dir);
		if (entry == NULL) {
			if (errno != 0) {
				LOG_ERROR("failed to read qemu log directory: %s", strerror(errno));
			}
			break;
		}

		if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
			continue;
		}
		n = snprintf(path, sizeof(path), "%s/%s", QEMU_LOG_DIR, entry->d_name);
		if (n < 0 || (size_t)n >= sizeof(path)) {
			LOG_ERROR("qemu log path is too long: %s", entry->d_name);
			continue;
		}
		dump_log_file(path);
	}
	closedir(dir);
}

static void cleanup_container_disks(const char *directory)
{
	DIR *dir = opendir(directory);
	struct dirent *entry;
	int directory_fd;

	if (dir == NULL) {
		if (errno != ENOENT) {
			LOG_ERROR("failed to open container disk directory %s: %s", directory, strerror(errno));
		}
		return;
	}
	directory_fd = dirfd(dir);
	if (directory_fd < 0) {
		LOG_ERROR("failed to access container disk directory %s: %s", directory, strerror(errno));
		closedir(dir);
		return;
	}
	for (;;) {
		size_t len;
		int saved_errno;

		errno = 0;
		entry = readdir(dir);
		if (entry == NULL) {
			if (errno != 0) {
				LOG_ERROR("failed to read container disk directory %s: %s", directory, strerror(errno));
			}
			break;
		}
		len = strlen(entry->d_name);

		if (len < 5 || strcmp(entry->d_name + len - 5, ".sock") != 0) {
			continue;
		}
		if (unlinkat(directory_fd, entry->d_name, 0) < 0 && errno != ENOENT) {
			saved_errno = errno;
			LOG_ERROR("failed to remove %s/%s: %s", directory, entry->d_name,
			    strerror(saved_errno));
		}
	}
	closedir(dir);
}

static int retry_delay_ms(int attempt)
{
	int delay = 10;
	int i;

	for (i = 0; i < attempt; i++) {
		delay *= 5;
	}
	return delay;
}

static int64_t monotonic_milliseconds(void)
{
	struct timespec now;

	if (clock_gettime(CLOCK_MONOTONIC, &now) < 0) {
		return -1;
	}
	return (int64_t)now.tv_sec * 1000 + now.tv_nsec / 1000000;
}

static int set_http_timeout(int fd, int option, int64_t deadline)
{
	int64_t now = monotonic_milliseconds();
	int64_t remaining;
	struct timeval timeout;

	if (now < 0) {
		return -errno;
	}
	remaining = deadline - now;
	if (remaining <= 0) {
		return -ETIMEDOUT;
	}
	timeout.tv_sec = remaining / 1000;
	timeout.tv_usec = (remaining % 1000) * 1000;
	return setsockopt(fd, SOL_SOCKET, option, &timeout, sizeof(timeout)) < 0 ? -errno : 0;
}

static int http_request(int port, const char *request, int *status, bool *envoy_server)
{
	int fd, err;
	struct sockaddr_in addr;
	char buf[2048];
	size_t off = 0, request_len = strlen(request), request_off = 0;
	ssize_t n;
	char *line, *saveptr, *hdr, *header_end;
	int64_t deadline;

	*status = 0;
	if (envoy_server != NULL) {
		*envoy_server = false;
	}

	deadline = monotonic_milliseconds();
	if (deadline < 0) {
		return -errno;
	}
	deadline += HTTP_TIMEOUT_SEC * 1000;

	memset(&addr, 0, sizeof(addr));
	addr.sin_family = AF_INET;
	addr.sin_port = htons((uint16_t)port);
	addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
	/* On Linux, SO_SNDTIMEO also bounds connect. Retry EINTR on a new
	 * socket because the interrupted connection's state is unspecified. */
	for (;;) {
		fd = socket(AF_INET, SOCK_STREAM | SOCK_CLOEXEC, 0);
		if (fd < 0) {
			return -errno;
		}
		err = set_http_timeout(fd, SO_SNDTIMEO, deadline);
		if (err < 0) {
			close(fd);
			return err;
		}
		if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == 0) {
			break;
		}
		err = -errno;
		close(fd);
		if (err == -EINTR) {
			continue;
		}
		return err == -EINPROGRESS ? -ETIMEDOUT : err;
	}

	while (request_off < request_len) {
		err = set_http_timeout(fd, SO_SNDTIMEO, deadline);
		if (err < 0) {
			goto done;
		}
		n = send(fd, request + request_off, request_len - request_off, MSG_NOSIGNAL);
		if (n > 0) {
			request_off += (size_t)n;
			continue;
		}
		if (n < 0 && errno == EINTR) {
			continue;
		}
		err = n == 0 ? -EPIPE : (errno == EAGAIN || errno == EWOULDBLOCK ? -ETIMEDOUT : -errno);
		goto done;
	}

	while (off < sizeof(buf) - 1) {
		err = set_http_timeout(fd, SO_RCVTIMEO, deadline);
		if (err < 0) {
			goto done;
		}
		n = recv(fd, buf + off, sizeof(buf) - 1 - off, 0);
		if (n < 0 && errno == EINTR) {
			continue;
		}
		if (n < 0) {
			err = errno == EAGAIN || errno == EWOULDBLOCK ? -ETIMEDOUT : -errno;
			goto done;
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
	buf[off] = '\0';
	header_end = strstr(buf, "\r\n\r\n");
	if (header_end == NULL) {
		header_end = strstr(buf, "\n\n");
	}
	if (header_end == NULL) {
		err = -EPROTO;
		goto done;
	}
	*header_end = '\0';

	line = strtok_r(buf, "\r\n", &saveptr);
	if (line == NULL || sscanf(line, "HTTP/%*s %d", status) != 1) {
		err = -EPROTO;
		goto done;
	}
	while (envoy_server != NULL && (hdr = strtok_r(NULL, "\r\n", &saveptr)) != NULL) {
		if (strncasecmp(hdr, "Server:", 7) == 0) {
			char *value = hdr + 7;
			char *end;

			while (*value == ' ' || *value == '\t') {
				value++;
			}
			end = value + strlen(value);
			while (end > value && (end[-1] == ' ' || end[-1] == '\t')) {
				end--;
			}
			*envoy_server = end - value == 5 && strncasecmp(value, "envoy", 5) == 0;
		}
	}
	err = 0;
done:
	close(fd);
	return err;
}

static bool is_retryable(int err)
{
	err = err < 0 ? -err : err;
	return err == ECONNREFUSED || err == ECONNRESET;
}

static bool istio_proxy_present(void)
{
	const char *req = "GET /healthz/ready HTTP/1.0\r\nHost: localhost\r\n\r\n";
	int attempt;

	for (attempt = 0; attempt < 5; attempt++) {
		int status = 0;
		bool envoy_server;
		int err = http_request(ENVOY_READY_PORT, req, &status, &envoy_server);

		if (err == 0 && envoy_server) {
			return true;
		}
		if (err < 0 && attempt < 4 && is_retryable(err)) {
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

	for (attempt = 0; attempt < 5; attempt++) {
		int status = 0;
		int err = http_request(ENVOY_QUIT_PORT, req, &status, NULL);
		bool retryable = (err == 0 && status == 503) || (err < 0 && is_retryable(err));

		if (err == 0 && status == 200) {
			return;
		}
		if (retryable) {
			if (attempt < 4) {
				sleep_ms(retry_delay_ms(attempt));
				continue;
			}
			LOG_ERROR("all attempts to terminate istio-proxy failed");
			return;
		}
		if (err == 0) {
			LOG_ERROR("Istio quit request returned HTTP %d", status);
			return;
		}
		LOG_ERROR("Istio quit request failed: %s", strerror(err < 0 ? -err : err));
		return;
	}
}

struct child_setup_error {
	int stage;
	int error;
};

enum child_setup_stage {
	CHILD_SIGNAL_MASK = 1,
	CHILD_CAPABILITY = 2,
	CHILD_EXEC = 3,
};

static void child_report_error(int fd, int stage, int error)
{
	struct child_setup_error report = { .stage = stage, .error = error };
	const char *data = (const char *)&report;
	size_t remaining = sizeof(report);

	while (remaining > 0) {
		ssize_t written = write(fd, data, remaining);
		if (written > 0) {
			data += written;
			remaining -= (size_t)written;
		} else if (written < 0 && errno == EINTR) {
			continue;
		} else {
			break;
		}
	}
	_exit(stage == CHILD_EXEC ? 127 : 1);
}

static const char *child_setup_stage_name(int stage)
{
	switch (stage) {
	case CHILD_SIGNAL_MASK:
		return "restore signal mask";
	case CHILD_CAPABILITY:
		return "raise CAP_NET_BIND_SERVICE";
	case CHILD_EXEC:
		return "exec virt-launcher";
	default:
		return "prepare virt-launcher";
	}
}

static int run_launcher(int argc, char **argv)
{
	char *child_argv[argc + 1];
	const char *launcher = DEFAULT_LAUNCHER;
	sigset_t block_chld, old_mask, run_mask, wait_signals, active_mask, suspend_mask;
	int exec_error_pipe[2];
	pid_t pid;
	struct child_setup_error child_error;
	size_t error_bytes = 0;
	int read_error = 0;
	int wait_failed = 0;

#ifdef VIRT_LAUNCHER_MONITOR_TESTING
	const char *test_launcher = getenv("VIRT_LAUNCHER");
	if (test_launcher != NULL && test_launcher[0] != '\0') {
		launcher = test_launcher;
	}
#endif

	if (pipe2(exec_error_pipe, O_CLOEXEC) < 0) {
		LOG_ERROR("failed to create launcher exec status pipe: %s", strerror(errno));
		return 1;
	}

	filter_launcher_args(argc, argv, child_argv);
	child_argv[0] = (char *)launcher;

	sigemptyset(&block_chld);
	sigaddset(&block_chld, SIGCHLD);
	if (sigprocmask(SIG_BLOCK, &block_chld, &old_mask) < 0) {
		LOG_ERROR("failed to block SIGCHLD: %s", strerror(errno));
		close(exec_error_pipe[0]);
		close(exec_error_pipe[1]);
		return 1;
	}
	if (install_sigchld_handler() < 0) {
		int err = errno;
		sigprocmask(SIG_SETMASK, &old_mask, NULL);
		close(exec_error_pipe[0]);
		close(exec_error_pipe[1]);
		LOG_ERROR("failed to install SIGCHLD handler: %s", strerror(err));
		return 1;
	}

	pid = fork();
	if (pid < 0) {
		int err = errno;
		sigprocmask(SIG_SETMASK, &old_mask, NULL);
		close(exec_error_pipe[0]);
		close(exec_error_pipe[1]);
		LOG_ERROR("failed to run %s: %s", launcher, strerror(err));
		return 1;
	}
	if (pid == 0) {
		close(exec_error_pipe[0]);
		if (sigprocmask(SIG_SETMASK, &old_mask, NULL) < 0) {
			child_report_error(exec_error_pipe[1], CHILD_SIGNAL_MASK, errno);
		}
#ifndef VIRT_LAUNCHER_MONITOR_TESTING
		if (raise_net_bind_capability() < 0) {
			child_report_error(exec_error_pipe[1], CHILD_CAPABILITY, errno);
		}
#endif
		execv(launcher, child_argv);
		child_report_error(exec_error_pipe[1], CHILD_EXEC, errno);
	}

	launcher_pid = pid;
	close(exec_error_pipe[1]);
	run_mask = old_mask;
	sigdelset(&run_mask, SIGCHLD);
	if (sigprocmask(SIG_SETMASK, &run_mask, NULL) < 0) {
		int err = errno;
		int status;
		pid_t waited;

		LOG_ERROR("failed to unblock SIGCHLD: %s", strerror(err));
		kill(pid, SIGTERM);
		close(exec_error_pipe[0]);
		do {
			waited = waitpid(pid, &status, 0);
		} while (waited < 0 && errno == EINTR);
		if (waited == pid) {
			record_launcher_status(status);
		} else {
			wait_error = 1;
		}
		return 1;
	}

	while (!read_error && error_bytes < sizeof(child_error)) {
		ssize_t n = read(exec_error_pipe[0], (char *)&child_error + error_bytes,
				 sizeof(child_error) - error_bytes);
		if (n > 0) {
			error_bytes += (size_t)n;
			continue;
		}
		if (n == 0) {
			break;
		}
		if (errno == EINTR) {
			continue;
		}
		read_error = errno;
		break;
	}
	close(exec_error_pipe[0]);
	if (read_error != 0) {
		LOG_ERROR("failed to read launcher exec status: %s", strerror(read_error));
		kill(pid, SIGTERM);
	} else if (error_bytes != 0) {
		if (error_bytes != sizeof(child_error)) {
			LOG_ERROR("received an incomplete launcher exec status");
		} else {
			LOG_ERROR("failed to %s: %s", child_setup_stage_name(child_error.stage),
			    strerror(child_error.error));
		}
		read_error = EIO;
	}

	sigemptyset(&wait_signals);
	sigaddset(&wait_signals, SIGCHLD);
	sigaddset(&wait_signals, SIGINT);
	sigaddset(&wait_signals, SIGTERM);
	sigaddset(&wait_signals, SIGQUIT);
	if (sigprocmask(SIG_BLOCK, &wait_signals, &active_mask) < 0) {
		LOG_ERROR("failed to block signals while waiting for virt-launcher: %s", strerror(errno));
		kill(launcher_pid, SIGTERM);
		return 1;
	}
	suspend_mask = active_mask;
	sigdelset(&suspend_mask, SIGCHLD);
	sigdelset(&suspend_mask, SIGINT);
	sigdelset(&suspend_mask, SIGTERM);
	sigdelset(&suspend_mask, SIGQUIT);

	while (launcher_exit_code < 0) {
		if (wait_error) {
			kill(launcher_pid, SIGTERM);
			wait_failed = 1;
			break;
		}
		if (termination_requests > 0) {
			termination_requests--;
			LOG("signalling virt-launcher to shut down");
			if (kill(launcher_pid, SIGTERM) < 0) {
				LOG_ERROR("received signal but can't signal virt-launcher to shut down: %s",
				    strerror(errno));
			}
			continue;
		}
		/* SIGCHLD cannot slip between the state check and this wait: all
		 * handled signals stay blocked until sigsuspend atomically unblocks them. */
		if (sigsuspend(&suspend_mask) < 0 && errno != EINTR) {
			LOG_ERROR("failed to wait for virt-launcher signal: %s", strerror(errno));
			kill(launcher_pid, SIGTERM);
			wait_failed = 1;
			break;
		}
	}
	if (sigprocmask(SIG_SETMASK, &active_mask, NULL) < 0) {
		LOG_ERROR("failed to restore signal mask: %s", strerror(errno));
		return 1;
	}
	if (read_error != 0 || wait_error || wait_failed) {
		return 1;
	}
	return launcher_exit_code;
}

int main(int argc, char **argv)
{
	const char *disk_dir = container_disk_dir_from_args(argc, argv);
	bool keep = keep_after_failure_from_args(argc, argv);
	int exit_code;
	int monitor_error = 0;

#ifdef VIRT_LAUNCHER_MONITOR_TESTING
	/* Bazel's Go test process is not PID 1; emulate PID-1 orphan adoption so
	 * the integration test can verify that the monitor reaps descendants. */
	if (prctl(PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0) < 0) {
		LOG_ERROR("failed to enable test child subreaper: %s", strerror(errno));
		return 1;
	}
#endif

	if (install_termination_handlers() < 0) {
		LOG_ERROR("failed to install signal handlers: %s", strerror(errno));
		return 1;
	}

	exit_code = run_launcher(argc, argv);
	if (exit_code != 0) {
		LOG_ERROR("dirty virt-launcher shutdown: exit-code %d", exit_code);
	}
	if (wait_error) {
		LOG_ERROR("waitpid failed while reaping child processes");
		monitor_error = 1;
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
