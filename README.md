# notify-send for WSL

This is a `notify-send` drop-in replacement for WSL processes that need to
show Windows notifications. It is designed to accept the command-line
interface used by `libnotify-bin`, with Windows toast delivery as the backend.

The WSL command is a shell wrapper. It forwards the request to the Windows
`notify-send.exe` backend through WSLInterop, so no Linux D-Bus notification
service is required.

## Design Goal: mise Conflict Notifications

The primary consumer is mise's [conflict notification
feature](https://mise.jdx.dev/history.html#conflict-notifications). On Linux,
mise looks up an executable named `notify-send` and constructs the command in
its [Linux notifier implementation](https://github.com/jdx/mise/blob/main/src/system/history/notify.rs),
specifically the `notifier` and `linux_notification` functions:

```text
notify-send --app-name mise --urgency normal [--icon <path>] -- <title> <body>
```

This project intentionally targets the `notify-send` command contract rather
than only mise. It does not provide a Linux D-Bus notification daemon.

## Requirements

- WSL2 with WSLInterop enabled
- Windows notifications enabled
- mise 2026.9 or later

## Build and Test

Install the locked project tools and run the no-project-install smoke test from WSL:

```bash
mise install --locked
mise run test-notification
```

This builds `notify-send.exe`, uses the bundled mise icon, and sends:

```bash
notify-send --app-name mise --urgency normal \
  --icon /path/to/wsl-notify-send/assets/mise.png -- \
  "Test notification" "WSLInterop is working"
```

To run the same test manually:

```bash
mise run build-windows
WSL_NOTIFY_SEND_EXE="$PWD/notify-send.exe" \
  ./bin/notify-send --app-name notify-send --urgency normal \
  --icon "$PWD/assets/speech-bubble.png" -- \
  "Test notification" "WSLInterop is working"
```

## Release

Authenticate the GitHub CLI, then run the release task from a clean, pushed
branch:

```bash
gh auth login
mise run release 1.2.3
```

The task creates and publishes the `v1.2.3` GitHub release with generated
release notes. The release workflow then builds and uploads the platform
assets.

## Install

Install the latest release directly into the global mise tool environment:

```bash
mise use -g github:dansailer/wsl-notify-send
```

The Linux release asset contains both the WSL `notify-send` wrapper and its
Windows `notify-send.exe` companion. The wrapper locates the companion from
the mise installation directory, so no separate Windows installation or
`WSL_NOTIFY_SEND_EXE` setting is required.

The corresponding release assets are `notify-send-linux-x64.tar.gz` and
`notify-send-windows-x64.zip`.

Verify the installed command:

```bash
command -v notify-send
notify-send --app-name mise --urgency normal -- \
  "Test notification" "Installed through mise"
```

For a source checkout, build the Windows backend and install the wrapper:

```bash
mise run build-windows
mise run install-wsl
```

Install the WSL wrapper in the user `PATH`:

```bash
mise run install-wsl
```

Install the pre-commit hook when desired:

```bash
mise run install-hooks
```

If the Windows executable is not on the imported WSL `PATH`, set its path:

```bash
export WSL_NOTIFY_SEND_EXE=/mnt/c/Tools/notify-send.exe
```

The wrapper must appear before any D-Bus-based Linux `notify-send`:

```bash
export PATH="$HOME/.local/bin:$PATH"
command -v notify-send
```

The first command should resolve to the wrapper, not `/usr/bin/notify-send`.

## Supported Interface

The reference interface is the `notify-send` implementation shipped by
[`libnotify-bin`](https://packages.debian.org/stable/libnotify-bin). The
authoritative option definitions are in libnotify's
[`tools/notify-send.c`](https://github.com/GNOME/libnotify/blob/main/tools/notify-send.c).

This project's parser and exit-code behavior are implemented in
[`main.go`](main.go); Windows delivery is implemented in
[`notify_windows.go`](notify_windows.go), icon staging is implemented in
[`icon.go`](icon.go), and WSL path conversion is handled by
[`bin/notify-send`](bin/notify-send).

```text
Usage: notify-send [OPTION...] <SUMMARY> [BODY]

  -u, --urgency=LEVEL
  -t, --expire-time=TIME
  -a, --app-name=APP_NAME
  -i, --icon=ICON
  -n, --app-icon=ICON
  -c, --category=CATEGORY
  -e, --transient
  -h, --hint=TYPE:NAME:VALUE
  -p, --print-id
      --id-fd=FD
  -r, --replace-id=ID
  -w, --wait
  -A, --action=[NAME=]TEXT
      --selected-action-fd=FD
      --activation-token-fd=FD
  -?, --help
  -v, --version
      --debug
```

The `--` separator is supported so summaries beginning with `-` are passed
unchanged. Existing WSL icon paths are converted to Windows paths by the
wrapper. Existing image files are then staged into the Windows temp directory
before the toast is submitted, because the Windows notification broker may not
be able to load an image directly from a WSL UNC path.

The compatibility behavior is:

- `--app-name`, `--icon`, `--app-icon`, `--urgency`, and `--expire-time` are mapped to the Windows toast where Windows provides an equivalent.
- `--category`, `--transient`, `--hint`, `--replace-id`, `--wait`, `--action`, `--selected-action-fd`, and `--activation-token-fd` are accepted so callers do not fail on unsupported D-Bus-specific options.
- `--print-id` and `--id-fd` return a process-generated ID. Windows toast IDs are not exposed by the backend, so `--replace-id` cannot replace an existing Windows toast exactly as it does through D-Bus.
- Actions and wait semantics cannot be identical without a persistent D-Bus notification server; the notification is still submitted successfully.

## Mise Integration

mise looks up `notify-send` in `PATH` and invokes it in this form:

```text
notify-send --app-name mise --urgency normal --icon <cache>/notifications/mise.png -- <title> <body>
```

On Linux, mise's default cache path is:

```text
${MISE_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/mise}/notifications/mise.png
```

Notifications are enabled by default in mise. They can be disabled with:

```bash
mise settings set history.notify false
```

## Exit Codes

- `0`: notification accepted by Windows
- `2`: invalid command-line arguments
- `3`: Windows notification backend failed
