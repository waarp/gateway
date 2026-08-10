#!/usr/bin/env bash
#
# Validates the packages produced by nFPM: policy linting, a real install on a
# minimal system, the resulting ownership and permissions, an actual daemon
# start, and the full install/upgrade/remove/purge lifecycle.
#
# Every check here exists because the corresponding defect shipped to users.
# See issue #387 and the notes in .claude/rules/packaging.md.
#
# Usage:
#   ./make.sh check-packages [deb|rpm|all]
#
# By default the script re-executes itself inside a disposable container: the
# lifecycle checks install and purge packages as root, which must never happen
# on the developer's machine. Set WAARP_CHECK_INNER=1 to run the checks
# directly in the current root filesystem — that is what CI does, since a CI
# job container is already disposable. Do not set it anywhere else.

set -uo pipefail

FAILURES=0
CHECKS=0

#####################################################################
###  ASSERTIONS
#####################################################################

# Checks never abort the run: the point is to obtain the full list of what is
# broken in one pass, not to stop at the first defect.
pass() {
  CHECKS=$((CHECKS + 1))
  printf '  \033[32mok\033[0m   %s\n' "$1"
}

fail() {
  CHECKS=$((CHECKS + 1))
  FAILURES=$((FAILURES + 1))
  printf '  \033[31mFAIL\033[0m %s\n' "$1"
  if [ -n "${2:-}" ]; then
    printf '       %s\n' "$2"
  fi
}

check() {
  local label=$1
  shift
  local out
  if out=$("$@" 2>&1); then
    pass "$label"
  else
    fail "$label" "${out//$'\n'/$'\n'       }"
  fi
}

# assert_mode <path> <expected mode> <expected owner:group>
assert_mode() {
  local path=$1 want_mode=$2 want_owner=$3
  local got
  got=$(stat -c '%a %U:%G' "$path" 2>/dev/null) || {
    fail "$path exists" "no such file or directory"
    return
  }
  if [ "$got" = "$want_mode $want_owner" ]; then
    pass "$path is $want_mode $want_owner"
  else
    fail "$path is $want_mode $want_owner" "got $got"
  fi
}

assert_contains() {
  local label=$1 haystack=$2 needle=$3
  case "$haystack" in
    *"$needle"*) pass "$label" ;;
    *) fail "$label" "expected to find: $needle" ;;
  esac
}

assert_empty() {
  local label=$1 value=$2
  if [ -z "$value" ]; then
    pass "$label"
  else
    fail "$label" "expected empty, got: $value"
  fi
}

section() {
  printf '\n\033[1m== %s\033[0m\n' "$1"
}

# Aborts rather than reporting a failed check: a missing package is not a defect
# in the package, it is the run being pointed at nothing. In CI it means the job
# started without the artifacts of its producer, and every check below would go
# red for a reason that has nothing to do with what they test.
require_package() {
  local path=$1 flavour=$2

  if [ ! -f "$path" ]; then
    echo "ERROR: no $flavour package at $path" >&2
    echo "       Build it first with './make.sh build dist && ./make.sh package $flavour'," >&2
    echo "       or point WAARP_DEB / WAARP_RPM at an existing file." >&2
    exit 2
  fi
}

#####################################################################
###  DEB CHECKS
#####################################################################

# Tolerated lintian tags. Each one is a deliberate decision, not an oversight.
#   statically-linked-binary          inherent to a CGO_ENABLED=0 Go binary
#   no-manual-page                    no man pages are written for this project
#   no-changelog                      the changelog lives in the documentation;
#                                     shipping a copy would add a fourth file to
#                                     keep in sync at every release (issue #387)
#   arch-dep-package-has-big-usr-share  the SNMP MIB and the example config
LINTIAN_TOLERATED=(
  statically-linked-binary
  no-manual-page
  no-changelog
  arch-dep-package-has-big-usr-share
)

lintian_suppressions() {
  local IFS=,
  echo "${LINTIAN_TOLERATED[*]}"
}

check_deb_lint() {
  section "lintian"

  local out
  out=$(lintian --tag-display-limit 0 --suppress-tags "$(lintian_suppressions)" \
    --fail-on error "$DEB" 2>&1)
  local status=$?

  echo "$out" | grep -v '^running with root' | sed 's/^/       /'

  if [ $status -eq 0 ]; then
    pass "lintian reports no error"
  else
    fail "lintian reports no error" "exit status $status"
  fi
}

check_deb_install() {
  section "install on a minimal system"

  # No repository is configured on purpose: a package that needs one cannot be
  # installed by "dpkg -i", which is what the documentation tells users to do.
  check "dpkg -i succeeds with no repository configured" dpkg -i "$DEB"

  local state
  state=$(dpkg-query -W -f '${Status}' waarp-gateway 2>/dev/null)
  if [ "$state" = "install ok installed" ]; then
    pass "package reaches the 'installed' state"
  else
    fail "package reaches the 'installed' state" "got: ${state:-not installed}"
  fi
}

# The expected contents are listed once and shared by both flavours. Keeping a
# list per format is how example_config.yaml went unshipped for years, and
# there is no sense in reproducing that shape inside the guard against it.
# HELPER_DIR is the one genuine difference: /usr/libexec on RPM, /usr/lib on
# Debian, where /usr/libexec is only allowed from Policy 4.7.0.
expected_paths() {
  local helper_dir=$1

  cat <<EOF
/usr/bin/waarp-gateway
/usr/bin/waarp-gatewayd
$helper_dir/get-remote
$helper_dir/updateconf
/usr/share/waarp-gateway/example_config.yaml
/usr/share/waarp-gateway/waarp-gateway.mib
/usr/share/doc/waarp-gateway/copyright
/usr/lib/systemd/system/waarp-gatewayd.service
/usr/lib/systemd/system/waarp-gateway-get-remote.service
/usr/lib/systemd/system/waarp-gateway-get-remote.timer
/etc/waarp-gateway/gatewayd.ini
/etc/default/waarp-gateway-get-remote
EOF
}

check_contents() {
  local listing=$1 helper_dir=$2
  section "contents"

  local path
  for path in $(expected_paths "$helper_dir"); do
    assert_contains "ships $path" "$listing" "$path"
  done

  # The old paths must stay owned by the package so that purge removes them.
  # That they hold a shim rather than the binary is asserted in check_shims:
  # a listing cannot tell the two apart.
  for path in /usr/share/waarp-gateway/get-remote /usr/share/waarp-gateway/updateconf; do
    assert_contains "$path is still owned by the package" "$listing" "$path"
  done

  assert_empty "no file installed through a /usr-merge alias" \
    "$(echo "$listing" | grep -E '^/(lib|bin|sbin|lib64)/' | tr '\n' ' ')"
}

# The helpers live in a different directory on each family, so the
# documentation tells users to invoke them by bare name rather than by absolute
# path. That only works if the PATH baked into the systemd unit matches where
# the package actually installed them -- two facts that live in two files and
# have nothing keeping them in step. This is the check that keeps the
# per-format split honest.
check_bare_name_resolution() {
  section "the helpers resolve by bare name"

  local unit=/usr/lib/systemd/system/waarp-gatewayd.service
  local unit_path
  unit_path=$(sed -n "s/.*PATH=\([^ ']*\).*/\1/p" "$unit" 2>/dev/null | head -1)

  if [ -z "$unit_path" ]; then
    fail "the service unit sets a PATH" "no PATH= found in $unit"
    return
  fi

  local name resolved
  for name in get-remote updateconf; do
    resolved=$(PATH="$unit_path" command -v "$name" 2>/dev/null)
    if [ -n "$resolved" ]; then
      pass "$name resolves to $resolved through the unit's PATH"
    else
      fail "$name resolves through the unit's PATH" "not found in: $unit_path"
    fi
  done
}

check_permissions() {
  section "ownership and permissions"

  # /etc must be group-writable: the daemon creates passphrase.aes, the cluster
  # override file, fw.json and get-file.list there, all as the waarp user. The
  # sticky bit stops that write permission from also allowing the daemon
  # account to unlink or rename root-owned files in the same directory.
  assert_mode /etc/waarp-gateway 1770 root:waarp
  assert_mode /etc/waarp-gateway/gatewayd.ini 640 root:waarp
  # root:root, not root:waarp: it holds the REST credentials and systemd reads
  # it as root before dropping privileges.
  assert_mode /etc/default/waarp-gateway-get-remote 640 root:root
  assert_mode /var/lib/waarp-gateway 750 waarp:waarp
  assert_mode /var/lib/waarp-gateway/db 750 waarp:waarp
  assert_mode /var/log/waarp-gateway 750 waarp:waarp
  # /usr/share holds no secret and must stay world-readable, otherwise the
  # shims below it are unreachable for anyone outside the waarp group.
  assert_mode /usr/share/waarp-gateway 755 root:root

  assert_empty "no file under /var/lib is executable" \
    "$(find /var/lib/waarp-gateway -type f -perm /0111 2>/dev/null | tr '\n' ' ')"
}

check_daemon_starts() {
  section "the daemon actually starts"

  # This is the check that a permissions mistake cannot survive: loadAESKey
  # runs before the database is even created, so a directory the daemon cannot
  # write to means the service never comes up, on every fresh install.
  #
  # Polled rather than run under a fixed "timeout 20": treating exit 124 as
  # success meant the passing case always cost the full twenty seconds, on
  # every local run and every pipeline.
  runuser -u waarp -- /usr/bin/waarp-gatewayd server \
    -c /etc/waarp-gateway/gatewayd.ini >/tmp/gwd.log 2>&1 &
  local pid=$!

  local waited=0
  while [ "$waited" -lt 40 ]; do
    if [ -s /etc/waarp-gateway/passphrase.aes ]; then
      break
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    sleep 0.5
    waited=$((waited + 1))
  done

  if [ -s /etc/waarp-gateway/passphrase.aes ]; then
    pass "the daemon creates its AES passphrase file"
  else
    fail "the daemon creates its AES passphrase file" \
      "$(tail -5 /tmp/gwd.log 2>/dev/null)"
  fi

  # Still running once it has written the key: it got past loadAESKey and
  # opened the database rather than crashing straight after.
  sleep 1
  if kill -0 "$pid" 2>/dev/null; then
    pass "the daemon stays up"
  else
    fail "the daemon stays up" "$(tail -5 /tmp/gwd.log 2>/dev/null)"
  fi

  kill "$pid" 2>/dev/null
  wait "$pid" 2>/dev/null
}

check_shims() {
  section "compatibility shims"

  local shim=/usr/share/waarp-gateway/updateconf
  local target=/usr/lib/waarp-gateway/updateconf

  if [ ! -x "$shim" ]; then
    fail "the shim is executable" "$shim is missing or not executable"
    return
  fi

  # The old path must hold a script, not the compiled helper left where it was:
  # a listing alone cannot tell the two apart, and asserting on the listing is
  # how this check would silently pass for the wrong reason.
  if [ "$(head -c2 "$shim")" = '#!' ]; then
    pass "the old path holds a script, not the compiled helper"
  else
    fail "the old path holds a script, not the compiled helper" \
      "$shim is still a binary"
    return
  fi

  if [ ! -x "$target" ]; then
    fail "the shim has a target to exec" "$target is missing"
    return
  fi

  # Stub the target so the assertions below measure the shim and nothing else.
  # Invoking the real helper would test its own argument handling instead.
  mv "$target" "$target.real"
  cat >"$target" <<'STUB'
#!/bin/sh
echo "argv:$*"
exit 3
STUB
  chmod 0755 "$target"

  local out err status
  out=$("$shim" "a b" --opt 2>/tmp/shim.err)
  status=$?
  err=$(cat /tmp/shim.err)

  mv "$target.real" "$target"

  # stdout must stay pristine: EXECMOVE reads its last line as the new file
  # path and EXECOUTPUT parses it for NEWFILENAME:.
  assert_contains "the deprecation warning goes to stderr" "$err" "deprecated"
  if [ "$out" = 'argv:a b --opt' ]; then
    pass "stdout carries the target's output alone, with argv intact"
  else
    fail "stdout carries the target's output alone, with argv intact" \
      "expected 'argv:a b --opt', got '$out'"
  fi

  # Waarp turns exit code 1 into a warning and anything else into a failure, so
  # the shim must propagate the real status rather than the shell's own.
  if [ "$status" -eq 3 ]; then
    pass "the shim propagates the exit status"
  else
    fail "the shim propagates the exit status" "expected 3, got $status"
  fi
}

check_lifecycle() {
  section "upgrade, remove and purge"

  dpkg -P waarp-gateway >/dev/null 2>&1

  # The upgrade is exercised from a real previous release when one is at hand,
  # which is the faithful case. In CI only the freshly built packages are
  # available, so the base install is the new package itself and the state the
  # old postinst used to leave behind is reproduced by hand below. What is
  # being tested either way is whether the new postinst repairs that state.
  if [ -f "$OLD_DEB" ]; then
    check "the previous release installs" dpkg -i "$OLD_DEB"
  else
    printf '  \033[33mskip\033[0m upgrading from a real previous release: %s is absent\n' \
      "$(basename "$OLD_DEB")"
    printf '       (pass WAARP_OLD_DEB to exercise it; the legacy state below is\n'
    printf '        reproduced by hand regardless)\n'
    check "the package installs" dpkg -i "$DEB"
  fi

  mkdir -p /var/lib/waarp-gateway/db
  echo fake-database >/var/lib/waarp-gateway/db/waarp-gateway.db
  echo fake-key >/etc/waarp-gateway/passphrase.aes
  # Reproduce the state the old postinst leaves behind, which dpkg never resets
  # and which therefore survives into the upgraded system.
  chmod -R 774 /var/lib/waarp-gateway
  chown -R root:waarp /usr/share/waarp-gateway 2>/dev/null
  chmod -R 750 /usr/share/waarp-gateway 2>/dev/null
  chown -R waarp:waarp /etc/waarp-gateway

  check "installing over the legacy state succeeds" dpkg -i "$DEB"

  assert_mode /usr/share/waarp-gateway 755 root:root
  assert_mode /etc/waarp-gateway 1770 root:waarp
  # Scoped to db/, which is what the postinst repairs. Files under in/, out/
  # and tmp/ are the administrator's, and their modes are deliberately left
  # alone even though the old "chmod -R 774" touched them too.
  assert_empty "the upgrade clears the legacy executable bits under db/" \
    "$(find /var/lib/waarp-gateway/db -type f -perm /0111 2>/dev/null | tr '\n' ' ')"

  # An unprivileged caller must be able to reach the shim, otherwise it fails
  # exactly on the upgraded hosts it exists for. The old postinst leaves
  # /usr/share/waarp-gateway at 0750 root:waarp and dpkg never resets an
  # existing directory, so only the new postinst can repair this.
  useradd -m -s /bin/sh joe 2>/dev/null
  local out status
  # "su -s" and not "runuser": runuser collapses an EACCES into exit 1, which
  # is indistinguishable from the helper itself failing, so the check could
  # never go red. Through a shell, a denied exec is unambiguously 126.
  out=$(su -s /bin/sh joe -c '/usr/share/waarp-gateway/updateconf --version' 2>&1)
  status=$?
  if [ "$status" -ne 126 ] && [ "${out#*Permission denied}" = "$out" ]; then
    pass "an unprivileged user can execute the shim after an upgrade"
  else
    fail "an unprivileged user can execute the shim after an upgrade" \
      "exit $status: $out"
  fi

  check "remove succeeds" dpkg -r waarp-gateway
  if [ -s /var/lib/waarp-gateway/db/waarp-gateway.db ]; then
    pass "the database survives 'remove'"
  else
    fail "the database survives 'remove'" "the file is gone"
  fi

  check "purge succeeds" dpkg -P waarp-gateway
  # The AES passphrase encrypts every stored credential and the PGP private
  # keys. Destroying it while the database survives is unrecoverable, and the
  # gateway restarts without a single error message afterwards.
  if [ -s /etc/waarp-gateway/passphrase.aes ]; then
    pass "the AES passphrase survives 'purge'"
  else
    fail "the AES passphrase survives 'purge'" "the key was destroyed"
  fi
}

run_deb_checks() {
  require_package "$DEB" deb
  install_linter deb
  printf '\033[1m### Debian package: %s\033[0m\n' "$DEB"

  check_deb_lint
  check_deb_install
  check_contents "$(dpkg -L waarp-gateway 2>/dev/null)" /usr/lib/waarp-gateway
  check_bare_name_resolution
  check_permissions
  check_daemon_starts
  check_shims
  check_lifecycle
}

#####################################################################
###  RPM CHECKS
#####################################################################

# Tolerated rpmlint tags, each a decision rather than an oversight.
#   no-binary                     pre-existing rpmlint quirk with nFPM packages;
#                                 0.12.9 reports it too, and "rpm -qlp" shows
#                                 the ELF binaries plainly under /usr/bin
#   no-group-tag                  the Group tag has been deprecated for years;
#                                 adding one back only to satisfy a linter
#   no-changelogname-tag          the changelog lives in the documentation, see
#                                 issue #387
#   not-listed-as-documentation   nFPM's content types are symlink, ghost,
#                                 config, dir and tree: there is no way to mark
#                                 a file %doc
#   non-standard-dir-perm         deliberate: the daemon's state must not be
#   non-readable                  world-readable, and gatewayd.ini carries the
#                                 RDBMS password on MySQL and PostgreSQL setups
#   dangerous-command-in-%pre     matches the literal word "install" in the
#                                 action guard, and the intentional chown
#   dangerous-command-in-%post    in the postinst
#   invalid-license               rpmlint on Rocky 9 predates SPDX identifiers
#   summary-not-capitalized       Debian requires a lowercase synopsis and RPM
#                                 a capitalized one; the field is shared
#   spelling-error                "interoperability" is not in rpmlint's dictionary
RPMLINT_TOLERATED=(
  no-binary
  no-group-tag
  no-changelogname-tag
  not-listed-as-documentation
  non-standard-dir-perm
  non-readable
  no-manual-page-for-binary
  no-documentation
  statically-linked-binary
  log-files-without-logrotate
  dangerous-command-in-
  invalid-license
  summary-not-capitalized
  spelling-error
)

run_rpm_checks() {
  require_package "$RPM" rpm
  install_linter rpm
  printf '\033[1m### RPM package: %s\033[0m\n' "$RPM"

  section "rpmlint"
  # Rocky 9 ships rpmlint 1.11, whose configuration is a Python file read with
  # -f. Neither "-o 'filters=[...]'" (the 2.x TOML form) nor "-o 'Filters
  # [...]'" has any effect there, and both fail silently -- the run simply
  # reports every tag as though no filter had been given.
  local rc=/tmp/rpmlintrc
  {
    echo "from Config import addFilter"
    local tag
    for tag in "${RPMLINT_TOLERATED[@]}"; do
      echo "addFilter(\"$tag\")"
    done
  } >"$rc"

  check "rpmlint reports no error" rpmlint -f "$rc" "$RPM"

  section "install"
  check "rpm -i succeeds" rpm -i "$RPM"

  check_contents "$(rpm -ql waarp-gateway 2>/dev/null)" /usr/libexec/waarp-gateway
  check_bare_name_resolution
  check_permissions

  section "the configuration is not overwritten on upgrade"
  # nFPM maps "type: config" to a bare %config, which RPM replaces on upgrade.
  # Only %config(noreplace) keeps the administrator's edits.
  local flags
  flags=$(rpm -qp --qf '[%{FILENAMES} %{FILEFLAGS:fflags}\n]' "$RPM" 2>/dev/null |
    grep '/etc/waarp-gateway/gatewayd.ini')
  assert_contains "gatewayd.ini is marked noreplace" "$flags" "n"
}

#####################################################################
###  CONTAINER PLUMBING
#####################################################################

# Resolved at top level rather than in a function: a "engine=$(container_cmd)"
# assignment runs the function in a subshell, where an "exit 2" for "no engine
# found" would kill only that subshell. The script then went on to run the
# empty string as a command and finished green having checked nothing.
resolve_engine() {
  if command -v podman >/dev/null 2>&1; then
    ENGINE=podman
  elif command -v docker >/dev/null 2>&1; then
    ENGINE=docker
  else
    echo "ERROR: neither podman nor docker is available" >&2
    exit 2
  fi
}

image_for() {
  case $1 in
    deb) echo docker.io/library/debian:stable-slim ;;
    rpm) echo docker.io/library/rockylinux:9 ;;
  esac
}

# Installs the linter the given flavour needs. Called from the inner run, so
# that the CI jobs -- which run the checks directly in their own container --
# get the same command rather than a second copy of it in .gitlab-ci.yml that
# can drift.
install_linter() {
  case $1 in
    deb)
      command -v lintian >/dev/null 2>&1 && return 0
      apt-get update -qq >/dev/null && apt-get install -y -qq lintian >/dev/null
      ;;
    rpm)
      command -v rpmlint >/dev/null 2>&1 && return 0
      dnf install -y -q rpmlint >/dev/null 2>&1
      ;;
  esac
}

run_in_container() {
  local flavour=$1
  local image
  image=$(image_for "$flavour")

  echo "==> running the $flavour checks in $image"
  # The repository is mounted read-only: the checks must never be able to
  # modify the working tree, only the container's own root filesystem.
  "$ENGINE" run --rm -v "$REPO_ROOT:/repo:ro" -e WAARP_CHECK_INNER=1 "$image" \
    /repo/dist/check-packages.sh "$flavour"
}

#####################################################################
###  MAIN
#####################################################################

# Derived from the script's own location, which is correct both in the
# container this script starts (where the repository is bind-mounted at /repo)
# and in a CI job, where it sits wherever the runner cloned it.
REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)

VERSION=$(tr -d '\n' <"$REPO_ROOT/VERSION")
DEB=${WAARP_DEB:-$REPO_ROOT/build/waarp-gateway_${VERSION}-1_amd64.deb}
RPM=${WAARP_RPM:-$REPO_ROOT/build/waarp-gateway-${VERSION}-1.x86_64.rpm}
OLD_DEB=${WAARP_OLD_DEB:-$REPO_ROOT/build/waarp-gateway_0.12.9-1_amd64.deb}

FLAVOUR=${1:-all}

case $FLAVOUR in
  deb | rpm) TARGETS=$FLAVOUR ;;
  all) TARGETS="deb rpm" ;;
  *)
    echo "ERROR: unknown flavour '$FLAVOUR' (expected deb, rpm or all)" >&2
    exit 2
    ;;
esac

if [ -z "${WAARP_CHECK_INNER:-}" ]; then
  resolve_engine
  status=0
  for target in $TARGETS; do
    run_in_container "$target" || status=1
  done
  exit $status
fi

if [ "$FLAVOUR" = all ]; then
  echo "ERROR: the inner run needs an explicit flavour, not 'all'" >&2
  exit 2
fi

case $FLAVOUR in
  deb) run_deb_checks ;;
  rpm) run_rpm_checks ;;
esac

printf '\n\033[1m%d checks, %d failures\033[0m\n' "$CHECKS" "$FAILURES"
[ "$FAILURES" -eq 0 ]
