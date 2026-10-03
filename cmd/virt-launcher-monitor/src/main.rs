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

use std::{
    ffi::{OsStr, OsString},
    fs::{self, File},
    io::{self, BufRead, BufReader},
    os::unix::process::CommandExt,
    path::{Path, PathBuf},
    process::{Command, Stdio},
    sync::atomic::{AtomicBool, AtomicI32, Ordering},
    thread,
    time::Duration,
};

use caps::{raise, CapSet, Capability};
use nix::{
    errno::Errno,
    sys::{
        signal::{kill, sigaction, SaFlags, SigAction, SigHandler, SigSet, Signal},
        wait::{waitpid, WaitPidFlag, WaitStatus},
    },
    unistd::Pid,
};
use pico_args::Arguments;

const LAUNCHER: &str = "/usr/bin/virt-launcher";
const DEFAULT_CONTAINER_DISK_DIR: &str = "/var/run/kubevirt/container-disks";
const VIRT_PRIVATE_DIR: &str = "/var/run/kubevirt-private";
const SERIAL_PORT: u8 = 0;
const ENVOY_READY_URL: &str = "http://localhost:15021/healthz/ready";
const ENVOY_QUIT_URL: &str = "http://localhost:15020/quitquitquit";
const LOG_LINE_LIMIT: usize = 512 * 1024;

static TERMINATION_SIGNAL: AtomicI32 = AtomicI32::new(0);
static SIGCHLD_RECEIVED: AtomicBool = AtomicBool::new(false);

#[derive(Debug)]
struct MonitorArgs {
    launcher_args: Vec<OsString>,
    container_disk_dir: PathBuf,
    keep_after_failure: bool,
    uid: String,
}

extern "C" fn handle_signal(signal: i32) {
    if signal == Signal::SIGCHLD as i32 {
        SIGCHLD_RECEIVED.store(true, Ordering::SeqCst);
    } else {
        TERMINATION_SIGNAL.store(signal, Ordering::SeqCst);
    }
}

fn install_signal_handlers() -> io::Result<()> {
    let action = SigAction::new(
        SigHandler::Handler(handle_signal),
        SaFlags::empty(),
        SigSet::empty(),
    );

    for signal in [Signal::SIGINT, Signal::SIGTERM, Signal::SIGQUIT, Signal::SIGCHLD] {
        unsafe { sigaction(signal, &action) }.map_err(io::Error::other)?;
    }
    Ok(())
}

fn filter_launcher_args(args: &[OsString]) -> Vec<OsString> {
    args.iter()
        .filter(|argument| argument.as_os_str() != OsStr::new("--keep-after-failure"))
        .cloned()
        .collect()
}

fn parse_arguments(raw_args: Vec<OsString>) -> MonitorArgs {
    let mut parser = Arguments::from_vec(raw_args.clone());
    let uid = parser
        .opt_value_from_str::<_, String>("--uid")
        .unwrap_or(None)
        .unwrap_or_default();
    let container_disk_dir = parser
        .opt_value_from_str::<_, String>("--container-disk-dir")
        .unwrap_or(None)
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from(DEFAULT_CONTAINER_DISK_DIR));
    let keep_after_failure = raw_args
        .iter()
        .any(|argument| argument.as_os_str() == OsStr::new("--keep-after-failure"));
    let _unknown_args = parser.finish();

    MonitorArgs {
        launcher_args: filter_launcher_args(&raw_args),
        container_disk_dir,
        keep_after_failure,
        uid,
    }
}

fn main() {
    let raw_args: Vec<OsString> = std::env::args_os().skip(1).collect();
    let monitor_args = parse_arguments(raw_args);
    let mut monitor_error = false;

    if let Err(error) = install_signal_handlers() {
        eprintln!("virt-launcher-monitor: failed to install signal handlers: {error}");
        std::process::exit(1);
    }

    start_serial_console_term_file(&monitor_args.uid);

    let exit_code = match run_launcher(&monitor_args.launcher_args) {
        Ok(code) => code,
        Err(error) => {
            monitor_error = true;
            eprintln!("virt-launcher-monitor: {error}");
            1
        }
    };

    dump_launcher_logs();
    if let Err(error) = cleanup_qemu() {
        monitor_error = true;
        eprintln!("virt-launcher-monitor: {error}");
    }
    terminate_istio_proxy();
    cleanup_container_disks(&monitor_args.container_disk_dir);
    remove_serial_console_term_file(Path::new(VIRT_PRIVATE_DIR), &monitor_args.uid);
    reap_children();

    if monitor_args.keep_after_failure && (monitor_error || exit_code != 0) {
        eprintln!("virt-launcher-monitor: keeping container alive after failure");
        loop {
            thread::park();
        }
    }

    std::process::exit(if monitor_error { 1 } else { exit_code });
}

fn run_launcher(args: &[OsString]) -> io::Result<i32> {
    let mut command = Command::new(LAUNCHER);
    command.args(args);
    command.stdout(Stdio::inherit()).stderr(Stdio::inherit());
    unsafe {
        command.pre_exec(|| {
            raise(None, CapSet::Inheritable, Capability::CAP_NET_BIND_SERVICE)
                .map_err(|error| io::Error::other(format!("failed to raise inheritable capability: {error}")))?;
            raise(None, CapSet::Ambient, Capability::CAP_NET_BIND_SERVICE)
                .map_err(|error| io::Error::other(format!("failed to raise ambient capability: {error}")))?;
            Ok(())
        });
    }

    let child = command.spawn().map_err(|error| {
        io::Error::new(error.kind(), format!("failed to run {LAUNCHER}: {error}"))
    })?;
    let launcher_pid = Pid::from_raw(child.id() as i32);
    let mut launcher_status = None;
    let mut termination_sent = false;

    loop {
        let requested_signal = TERMINATION_SIGNAL.swap(0, Ordering::SeqCst);
        if requested_signal != 0 && !termination_sent {
            eprintln!(
                "virt-launcher-monitor: received signal {requested_signal}, signalling virt-launcher to shut down"
            );
            kill(launcher_pid, Signal::SIGTERM).map_err(io::Error::other)?;
            termination_sent = true;
        }

        reap_children_until_empty(&mut launcher_status, launcher_pid)?;
        if let Some(code) = launcher_status {
            eprintln!("virt-launcher-monitor: virt-launcher exited with code {code}");
            return Ok(code);
        }

        if SIGCHLD_RECEIVED.swap(false, Ordering::SeqCst) {
            continue;
        }
        thread::sleep(Duration::from_millis(10));
    }
}

fn reap_children_until_empty(
    launcher_status: &mut Option<i32>,
    launcher_pid: Pid,
) -> io::Result<()> {
    loop {
        match waitpid(Pid::from_raw(-1), Some(WaitPidFlag::WNOHANG)) {
            Ok(WaitStatus::StillAlive) => return Ok(()),
            Ok(WaitStatus::Exited(pid, code)) => {
                if pid == launcher_pid {
                    *launcher_status = Some(code);
                }
            }
            Ok(WaitStatus::Signaled(pid, signal, _)) => {
                if pid == launcher_pid {
                    *launcher_status = Some(128 + signal as i32);
                }
            }
            Ok(WaitStatus::Stopped(_, _))
            | Ok(WaitStatus::Continued(_))
            | Ok(WaitStatus::PtraceEvent(_, _, _))
            | Ok(WaitStatus::PtraceSyscall(_)) => {}
            Err(Errno::ECHILD) => {
                if launcher_status.is_none() {
                    return Err(io::Error::other("virt-launcher was not reaped"));
                }
                return Ok(());
            }
            Err(error) => return Err(io::Error::other(error)),
        }
    }
}

fn reap_children() {
    loop {
        match waitpid(Pid::from_raw(-1), Some(WaitPidFlag::WNOHANG)) {
            Ok(WaitStatus::StillAlive) | Err(Errno::ECHILD) => return,
            Ok(_) => {}
            Err(error) => {
                eprintln!("virt-launcher-monitor: failed to reap child: {error}");
                return;
            }
        }
    }
}

fn dump_launcher_logs() {
    dump_log_file(Path::new("/var/run/kubevirt/passt.log"));

    let directory = Path::new("/run/kubevirt-private/libvirt/qemu/log");
    let entries = match fs::read_dir(directory) {
        Ok(entries) => entries,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return,
        Err(error) => {
            eprintln!(
                "virt-launcher-monitor: failed to read {}: {error}",
                directory.display()
            );
            return;
        }
    };

    for entry in entries.flatten() {
        dump_log_file(&entry.path());
    }
}

fn dump_log_file(path: &Path) {
    let file = match File::open(path) {
        Ok(file) => file,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return,
        Err(error) => {
            eprintln!(
                "virt-launcher-monitor: failed to open {}: {error}",
                path.display()
            );
            return;
        }
    };

    eprintln!("virt-launcher-monitor: dump log file: {}", path.display());
    let mut reader = BufReader::new(file);
    let mut line = String::new();
    loop {
        line.clear();
        match reader.read_line(&mut line) {
            Ok(0) => break,
            Ok(_) => {
                if line.len() > LOG_LINE_LIMIT {
                    let limit = line
                        .char_indices()
                        .map(|(index, _)| index)
                        .take_while(|index| *index <= LOG_LINE_LIMIT)
                        .last()
                        .unwrap_or(0);
                    line.truncate(limit);
                    line.push_str(" [line truncated]");
                }
                eprint!("virt-launcher-monitor: {}: {}", path.display(), line);
                if !line.ends_with('\n') {
                    eprintln!();
                }
            }
            Err(error) => {
                eprintln!(
                    "virt-launcher-monitor: failed to read {}: {error}",
                    path.display()
                );
                break;
            }
        }
    }
}

fn cleanup_qemu() -> io::Result<()> {
    let pid = find_qemu_pid("qemu-system")?.or(find_qemu_pid("qemu-kvm")?);
    let Some(pid) = pid else { return Ok(()) };
    eprintln!("virt-launcher-monitor: killing QEMU gracefully (pid {pid})");
    kill(pid, Signal::SIGTERM).map_err(io::Error::other)?;

    for _ in 0..100 {
        reap_children();
        if !pid_exists(pid) {
            return Ok(());
        }
        thread::sleep(Duration::from_millis(100));
    }

    Err(io::Error::new(
        io::ErrorKind::TimedOut,
        "QEMU did not exit within 10 seconds",
    ))
}

fn find_qemu_pid(prefix: &str) -> io::Result<Option<Pid>> {
    let entries = match fs::read_dir("/proc") {
        Ok(entries) => entries,
        Err(_) => return Ok(None),
    };
    for entry in entries.flatten() {
        let name = entry.file_name();
        let Some(name) = name.to_str() else { continue };
        if !name.bytes().all(|byte| byte.is_ascii_digit()) {
            continue;
        }
        let cmdline = match fs::read(entry.path().join("cmdline")) {
            Ok(cmdline) => cmdline,
            Err(_) => continue,
        };
        if cmdline_matches(&cmdline, prefix) {
            if let Ok(raw_pid) = name.parse::<i32>() {
                return Ok(Some(Pid::from_raw(raw_pid)));
            }
        }
    }
    Ok(None)
}

fn cmdline_matches(cmdline: &[u8], needle: &str) -> bool {
    cmdline
        .windows(needle.len())
        .any(|window| window == needle.as_bytes())
}

fn pid_exists(pid: Pid) -> bool {
    kill(pid, None).is_ok()
}

fn serial_term_path(private_dir: &Path, uid: &str, suffix: &str) -> PathBuf {
    private_dir
        .join(uid)
        .join(format!("virt-serial{SERIAL_PORT}-log-sigTerm{suffix}"))
}

fn create_serial_console_term_file(private_dir: &Path, uid: &str, suffix: &str) -> io::Result<bool> {
    if uid.is_empty() {
        return Ok(false);
    }
    let path = serial_term_path(private_dir, uid, suffix);
    if path.exists() {
        return Ok(false);
    }
    File::create(&path)?;
    eprintln!(
        "virt-launcher-monitor: serial console term file created: {}",
        path.display()
    );
    Ok(true)
}

fn start_serial_console_term_file(uid: &str) {
    if uid.is_empty() {
        return;
    }
    let uid = uid.to_string();
    thread::spawn(move || {
        let private_dir = Path::new(VIRT_PRIVATE_DIR);
        for _ in 0..100 {
            match create_serial_console_term_file(private_dir, &uid, "") {
                Ok(true) => return,
                Ok(false) if serial_term_path(private_dir, &uid, "").exists() => return,
                _ => thread::sleep(Duration::from_millis(100)),
            }
        }
        eprintln!("virt-launcher-monitor: could not create serial console term file");
    });
}

fn remove_serial_console_term_file(private_dir: &Path, uid: &str) {
    if uid.is_empty() {
        return;
    }
    let path = serial_term_path(private_dir, uid, "");
    match fs::remove_file(&path) {
        Ok(()) => eprintln!(
            "virt-launcher-monitor: serial console term file deleted: {}",
            path.display()
        ),
        Err(error) if error.kind() == io::ErrorKind::NotFound => {}
        Err(error) => eprintln!(
            "virt-launcher-monitor: could not delete serial console term file {}: {error}",
            path.display()
        ),
    }
    if let Err(error) = create_serial_console_term_file(private_dir, uid, "-done") {
        eprintln!(
            "virt-launcher-monitor: could not create serial console term-done file: {error}"
        );
    }
}

fn cleanup_container_disks(directory: &Path) {
    let entries = match fs::read_dir(directory) {
        Ok(entries) => entries,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension() == Some(OsStr::new("sock")) {
            if let Err(error) = fs::remove_file(&path) {
                eprintln!(
                    "virt-launcher-monitor: failed to remove {}: {error}",
                    path.display()
                );
            }
        }
    }
}

fn retry_delay(attempt: usize) -> Duration {
    let mut delay = Duration::from_millis(10);
    for _ in 0..=attempt {
        delay *= 5;
    }
    delay
}

fn is_retryable_transport(error: &ureq::Error) -> bool {
    matches!(
        error,
        ureq::Error::Transport(transport)
            if matches!(
                transport.kind(),
                ureq::ErrorKind::ConnectionFailed | ureq::ErrorKind::Io
            )
    )
}

fn terminate_istio_proxy() {
    let agent = ureq::AgentBuilder::new()
        .timeout_connect(Duration::from_secs(2))
        .timeout_read(Duration::from_secs(2))
        .build();

    let mut envoy_present = false;
    for attempt in 0..4 {
        match agent.get(ENVOY_READY_URL).call() {
            Ok(response) => {
                envoy_present = response.header("server") == Some("envoy");
                break;
            }
            Err(error) if attempt < 3 && is_retryable_transport(&error) => {
                thread::sleep(retry_delay(attempt));
            }
            Err(_) => break,
        }
    }
    if !envoy_present {
        return;
    }

    for attempt in 0..4 {
        match agent.post(ENVOY_QUIT_URL).call() {
            Ok(response) if response.status() == 200 => return,
            Ok(response) if response.status() == 503 => {}
            Ok(response) => {
                eprintln!(
                    "virt-launcher-monitor: Istio quit request returned HTTP {}",
                    response.status()
                );
                return;
            }
            Err(ureq::Error::Status(503, _)) if attempt < 3 => {}
            Err(ureq::Error::Status(503, _)) => {
                eprintln!("virt-launcher-monitor: Istio quit request returned HTTP 503");
                return;
            }
            Err(error) if attempt < 3 && is_retryable_transport(&error) => {}
            Err(error) => {
                eprintln!("virt-launcher-monitor: Istio quit request failed: {error}");
                return;
            }
        }
        if attempt < 3 {
            thread::sleep(retry_delay(attempt));
        }
    }
    eprintln!("virt-launcher-monitor: all attempts to terminate Istio proxy failed");
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{SystemTime, UNIX_EPOCH};

    struct TempDirGuard(PathBuf);

    impl Drop for TempDirGuard {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }

    #[test]
    fn filters_only_keep_after_failure() {
        let args = vec![
            OsString::from("--uid"),
            OsString::from("vmi-uid"),
            OsString::from("--container-disk-dir=/tmp/disks"),
            OsString::from("--keep-after-failure"),
            OsString::from("-v"),
            OsString::from("--unknown"),
        ];
        let filtered = filter_launcher_args(&args);
        assert!(!filtered.iter().any(|arg| arg == "--keep-after-failure"));
        assert!(filtered.iter().any(|arg| arg == "--uid"));
        assert!(filtered.iter().any(|arg| arg == "vmi-uid"));
        assert!(filtered
            .iter()
            .any(|arg| arg == "--container-disk-dir=/tmp/disks"));
        assert!(filtered.iter().any(|arg| arg == "-v"));
        assert!(filtered.iter().any(|arg| arg == "--unknown"));
    }

    #[test]
    fn parses_keep_after_failure_and_disk_directory() {
        let parsed = parse_arguments(vec![
            OsString::from("--keep-after-failure"),
            OsString::from("--container-disk-dir"),
            OsString::from("/tmp/disks"),
        ]);
        assert!(parsed.keep_after_failure);
        assert_eq!(parsed.container_disk_dir, PathBuf::from("/tmp/disks"));
        assert!(parsed.uid.is_empty());
        assert!(!parse_arguments(Vec::new()).keep_after_failure);
    }

    #[test]
    fn parses_uid() {
        let parsed = parse_arguments(vec![
            OsString::from("--uid"),
            OsString::from("vmi-uid"),
        ]);
        assert_eq!(parsed.uid, "vmi-uid");
    }

    #[test]
    fn serial_term_paths_match_go_monitor() {
        let private_dir = Path::new("/var/run/kubevirt-private");
        assert_eq!(
            serial_term_path(private_dir, "vmi-uid", ""),
            PathBuf::from("/var/run/kubevirt-private/vmi-uid/virt-serial0-log-sigTerm")
        );
        assert_eq!(
            serial_term_path(private_dir, "vmi-uid", "-done"),
            PathBuf::from("/var/run/kubevirt-private/vmi-uid/virt-serial0-log-sigTerm-done")
        );
    }

    #[test]
    fn create_and_remove_serial_console_term_files() {
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("clock before epoch")
            .as_nanos();
        let private_dir = std::env::temp_dir().join(format!("virt-launcher-monitor-term-{unique}"));
        let uid_dir = private_dir.join("vmi-uid");
        fs::create_dir_all(&uid_dir).expect("create uid directory");
        let _guard = TempDirGuard(private_dir.clone());

        assert!(create_serial_console_term_file(&private_dir, "vmi-uid", "")
            .expect("create term file"));
        let term_path = serial_term_path(&private_dir, "vmi-uid", "");
        assert!(term_path.exists());
        assert!(!create_serial_console_term_file(&private_dir, "vmi-uid", "")
            .expect("existing term file"));

        remove_serial_console_term_file(&private_dir, "vmi-uid");
        assert!(!term_path.exists());
        assert!(serial_term_path(&private_dir, "vmi-uid", "-done").exists());
    }

    #[test]
    fn removes_only_socket_files() {
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("clock before epoch")
            .as_nanos();
        let directory = std::env::temp_dir().join(format!("virt-launcher-monitor-{unique}"));
        fs::create_dir(&directory).expect("create test directory");
        let _guard = TempDirGuard(directory.clone());
        fs::write(directory.join("disk.sock"), b"").expect("create socket file");
        fs::write(directory.join("disk.img"), b"").expect("create image file");
        cleanup_container_disks(&directory);
        assert!(!directory.join("disk.sock").exists());
        assert!(directory.join("disk.img").exists());
    }

    #[test]
    fn matches_qemu_command_lines() {
        assert!(cmdline_matches(b"qemu-system-x86_64\0-machine\0", "qemu-system"));
        assert!(cmdline_matches(b"/usr/libexec/qemu-kvm\0", "qemu-kvm"));
        assert!(!cmdline_matches(b"qemu-img\0convert\0", "qemu-system"));
        assert!(!cmdline_matches(b"qemu-img\0convert\0", "qemu-kvm"));
    }
}
