# Runs inside a built image via /bin/sh -c. $1 is the entrypoint binary.
# Exit 0 means pass or skip, exit 1 means the binary will not load.
# Kept in sync with the inline check in circleci/verify-linkage.
BIN="$1"
command -v ldd >/dev/null 2>&1 || { echo "SKIP: no ldd in image"; exit 0; }
P=$(command -v "$BIN" 2>/dev/null) || { echo "FAIL: entrypoint $BIN not found in image"; exit 1; }
head -c 4 "$P" | grep -q ELF || { echo "SKIP: $P is not an ELF binary"; exit 0; }
rc=0
out=$(ldd "$P" 2>&1) || rc=$?
echo "$out"
case "$out" in
  *"not a dynamic executable"*|*"Not a valid dynamic program"*)
    echo "OK: $P is statically linked"; exit 0 ;;
esac
[ "$rc" -eq 0 ] || { echo "FAIL: ldd exited $rc for $P"; exit 1; }
if echo "$out" | grep -q "not found"; then
  echo "FAIL: $P has unresolved shared libraries or symbol versions"; exit 1
fi
echo "OK: $P resolves all shared libraries against the runtime image"
