#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
wrapper="$root_dir/bin/notify-send"
bash_path="$(command -v bash)"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

backend="$temp_dir/notify-send.exe"
record="$temp_dir/args"
fake_bin="$temp_dir/fake-bin"
utility_bin="$temp_dir/utility-bin"
mkdir -p "$fake_bin" "$utility_bin"
ln -s "$(command -v readlink)" "$utility_bin/readlink"
ln -s "$(command -v dirname)" "$utility_bin/dirname"

printf '%s\n' \
  '#!/usr/bin/env bash' \
  'set -euo pipefail' \
  ': "${WRAPPER_TEST_RECORD:?}"' \
  'printf "%s\\0" "$@" > "$WRAPPER_TEST_RECORD"' \
  'exit "${WRAPPER_TEST_EXIT:-0}"' > "$backend"
chmod +x "$backend"

printf '%s\n' \
  '#!/usr/bin/env bash' \
  'set -euo pipefail' \
  '[[ "${1:-}" == "-w" ]]' \
  'printf "%s\\n" "C:\\converted\\icon.png"' > "$fake_bin/wslpath"
chmod +x "$fake_bin/wslpath"
cp "$backend" "$fake_bin/notify-send.exe"

reset_record() {
	: > "$record"
}

assert_args() {
	local label="$1"
	shift
	local -a actual=()
	mapfile -d '' actual < "$record"
	if [[ "${#actual[@]}" -ne "$#" ]]; then
		printf '%s: got %d arguments, want %d\n' "$label" "${#actual[@]}" "$#" >&2
		printf 'actual: %q\n' "${actual[@]}" >&2
		return 1
	fi

	local index=0 expected
	for expected in "$@"; do
		if [[ "${actual[index]}" != "$expected" ]]; then
			printf '%s: argument %d = %q, want %q\n' "$label" "$index" "${actual[index]}" "$expected" >&2
			return 1
		fi
		((index += 1))
	done
}

run_wrapper() {
	reset_record
	WRAPPER_TEST_RECORD="$record" "$@"
}

icon="$temp_dir/icon.png"
printf 'icon' > "$icon"
converted_icon='C:\converted\icon.png'

run_wrapper env \
	WSL_NOTIFY_SEND_EXE="$backend" \
	PATH="$fake_bin:$PATH" \
	"$bash_path" "$wrapper" \
	--app-name 'app name' --icon "$icon" -- '-summary' 'body with spaces'
assert_args 'explicit backend' \
	--app-name 'app name' --icon "$converted_icon" -- '-summary' 'body with spaces'

run_wrapper env \
	WSL_NOTIFY_SEND_EXE="$backend" \
	PATH="$fake_bin:$PATH" \
	"$bash_path" "$wrapper" \
	--icon warning Title
assert_args 'stock icon' --icon warning Title

run_wrapper env \
	WSL_NOTIFY_SEND_EXE="$backend" \
	PATH="$fake_bin:$PATH" \
	"$bash_path" "$wrapper" \
	-i "$icon" --app-icon="$icon" -n "$icon" -i"$icon" -n"$icon" Title
assert_args 'icon forms' \
	-i "$converted_icon" --app-icon="$converted_icon" -n "$converted_icon" \
	-i"$converted_icon" -n"$converted_icon" Title

companion_dir="$temp_dir/companion"
mkdir -p "$companion_dir"
cp "$wrapper" "$companion_dir/notify-send"
cp "$backend" "$companion_dir/notify-send.exe"
run_wrapper env PATH="$fake_bin:$PATH" "$bash_path" "$companion_dir/notify-send" Title
assert_args 'companion backend' Title

run_wrapper env \
	WSL_NOTIFY_SEND_EXE= \
	PATH="$fake_bin:$PATH" \
	"$bash_path" "$wrapper" Title
assert_args 'PATH fallback' Title

if WRAPPER_TEST_RECORD="$record" WRAPPER_TEST_EXIT=17 \
	env WSL_NOTIFY_SEND_EXE="$backend" PATH="$fake_bin:$PATH" \
	"$bash_path" "$wrapper" Title; then
	printf 'backend failure was not propagated\n' >&2
	exit 1
else
	status=$?
	if [[ "$status" -ne 17 ]]; then
		printf 'backend exit code = %d, want 17\n' "$status" >&2
		exit 1
	fi
fi

missing_stderr="$temp_dir/missing.stderr"
if env WSL_NOTIFY_SEND_EXE= PATH="$utility_bin" "$bash_path" "$wrapper" Title 2>"$missing_stderr"; then
	printf 'missing backend unexpectedly succeeded\n' >&2
	exit 1
else
	status=$?
	if [[ "$status" -ne 127 ]] || ! grep -q 'Windows backend not found' "$missing_stderr"; then
		printf 'missing backend status = %d, stderr = %s\n' "$status" "$(<"$missing_stderr")" >&2
		exit 1
	fi
fi

printf 'wrapper integration tests passed\n'
