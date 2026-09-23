# Runs inside a built image via /bin/sh -c, with no network. $1 is the
# entrypoint binary. Exit 0 means pass or skip, exit 1 means the binary
# will not start. Kept in sync with the inline check in
# circleci/verify-linkage.
BIN="$1"
P=$(command -v "$BIN" 2>/dev/null) || { echo "FAIL: entrypoint $BIN not found in image"; exit 1; }
head -c 4 "$P" | grep -q ELF || { echo "SKIP: $P is not an ELF binary"; exit 0; }

# 1. Load check: every shared library and glibc symbol version resolves.
if command -v ldd >/dev/null 2>&1; then
  rc=0
  out=$(ldd "$P" 2>&1) || rc=$?
  echo "$out"
  case "$out" in
    *"not a dynamic executable"*|*"Not a valid dynamic program"*)
      echo "OK: $P is statically linked" ;;
    *)
      [ "$rc" -eq 0 ] || { echo "FAIL: ldd exited $rc for $P"; exit 1; }
      if echo "$out" | grep -q "not found"; then
        echo "FAIL: $P has unresolved shared libraries or symbol versions"; exit 1
      fi
      echo "OK: $P resolves all shared libraries against the runtime image" ;;
  esac
else
  echo "SKIP load check: no ldd in image"
fi

# 2. Exec check: start the binary with no config for a few seconds. Most
# services exit early complaining about missing config, which is fine.
# Fail only on signatures that mean the binary can never start: loader
# or architecture errors, a panic during package init, or a native crash.
command -v timeout >/dev/null 2>&1 || { echo "SKIP exec check: no timeout in image"; exit 0; }
rc=0
out=$(timeout 5 "$P" </dev/null 2>&1) || rc=$?
case "$rc" in
  124|143) echo "OK: $P was still running after 5s"; exit 0 ;;
esac
sig=""
case "$rc" in
  126|127) sig="could not be executed (exit $rc)" ;;
esac
if echo "$out" | grep -qiE "error while loading shared libraries|exec format error|cannot execute binary"; then
  sig="failed to load (wrong architecture or missing libraries)"
fi
if echo "$out" | grep -q "runtime.doInit"; then
  sig="panicked during package init, before main"
fi
if echo "$out" | grep -qE "SIGILL|illegal instruction|fatal error: unexpected signal"; then
  sig="crashed with a fatal signal"
fi
if [ -z "$sig" ] && [ "$rc" -gt 128 ]; then
  sig="was killed by signal $((rc - 128))"
fi
if [ -n "$sig" ]; then
  echo "$out" | tail -n 30
  echo "FAIL: $P $sig"
  exit 1
fi
echo "OK: $P exited $rc without a crash signature (expected without config)"
