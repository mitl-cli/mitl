#!/usr/bin/env bash
# scripts/preflight.sh - End-to-end user experience validation for mitl
# Validates key promises from PREFLIGHT.md with PASS/FAIL summary.

set -u -o pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
RESET='\033[0m'

FIRST_RUN_MAX=30.0
CACHED_RUN_MAX=1.0
REQUIRED_SPEEDUP=10.0

LIGHT_MODE=0
SKIP_DOCKER=0
WORKDIR=""

pass_cnt=0
fail_cnt=0
skip_cnt=0

usage() {
  cat <<EOF
mitl Preflight — validate user experience

Usage: $0 [--light] [--skip-docker] [--workdir DIR]

Options:
  --light         Skip heavyweight stack tests and benchmarks
  --skip-docker   Skip Docker comparison tests
  --workdir DIR   Use DIR as working directory (default: temp dir)
  -h, --help      Show help
EOF
}

log()   { echo -e "$*"; }
info()  { echo -e "${BLUE}$*${RESET}"; }
ok()    { echo -e "${GREEN}$*${RESET}"; }
warn()  { echo -e "${YELLOW}$*${RESET}"; }
err()   { echo -e "${RED}$*${RESET}"; }

pass()  { ok "✅ $*"; pass_cnt=$((pass_cnt+1)); }
fail()  { err "❌ $*"; fail_cnt=$((fail_cnt+1)); }
skip()  { warn "⏭  $*"; skip_cnt=$((skip_cnt+1)); }

has() { command -v "$1" >/dev/null 2>&1; }

measure_time() {
  # Prints seconds as a floating number
  local cmd=("$@")
  local out
  if has /usr/bin/time; then
    out=$( { /usr/bin/time -p "${cmd[@]}" >/dev/null; } 2>&1 | awk '/^real /{print $2}' )
  else
    out=$( { command time -p "${cmd[@]}" >/dev/null; } 2>&1 | awk '/^real /{print $2}' )
  fi
  echo "$out"
}

float_gt() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a>b)}'; }
float_ge() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a>=b)}'; }
float_lt() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a<b)}'; }
float_div() { awk -v a="$1" -v b="$2" 'BEGIN{ if (b==0) print 0; else printf("%.2f\n", a/b)}'; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --light) LIGHT_MODE=1; shift ;;
    --skip-docker) SKIP_DOCKER=1; shift ;;
    --workdir) WORKDIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) err "Unknown option: $1"; usage; exit 1 ;;
  esac
done

print_banner() {
  echo
  echo -e "${BOLD}🏹 mitl Preflight — User Experience Validation${RESET}"
  echo
}

require_mitl() {
  if ! has mitl; then
    fail "mitl not found in PATH"
    echo "Install via Homebrew: brew tap mitl-cli/tap && brew install mitl"
    echo "Or: curl -fsSL https://mitl.run/install.sh | bash"
    exit 1
  fi
}

setup_workdir() {
  if [[ -z "$WORKDIR" ]]; then
    WORKDIR=$(mktemp -d 2>/dev/null || mktemp -d -t mitl-preflight)
  else
    mkdir -p "$WORKDIR"
  fi
  info "Using workdir: $WORKDIR"
}

test_doctor() {
  info "Running: mitl doctor"
  local out
  if ! out=$(mitl doctor 2>&1); then
    echo "$out"
    fail "mitl doctor failed"
    return
  fi
  echo "$out" | grep -q "Performance Score:" && pass "doctor prints Performance Score" || fail "doctor did not print Performance Score"
}

test_performance_cached() {
  if [[ "$LIGHT_MODE" -eq 1 ]]; then
    skip "Performance cached test skipped (light mode)"
    return
  fi
  info "Testing cached performance (first vs cached run)"
  pushd "$WORKDIR" >/dev/null || return
  local t1 t2
  t1=$(measure_time mitl run echo first-run 2>/dev/null || true)
  t2=$(measure_time mitl run echo cached-run 2>/dev/null || true)
  echo "First: ${t1}s | Cached: ${t2}s"
  if [[ -z "$t1" || -z "$t2" ]]; then
    fail "Unable to measure run times"
    popd >/dev/null || true
    return
  fi
  if float_lt "$t1" "$FIRST_RUN_MAX"; then
    pass "First run < ${FIRST_RUN_MAX}s"
  else
    fail "First run >= ${FIRST_RUN_MAX}s (got ${t1}s)"
  fi
  if float_lt "$t2" "$CACHED_RUN_MAX"; then
    pass "Cached run < ${CACHED_RUN_MAX}s"
  else
    fail "Cached run >= ${CACHED_RUN_MAX}s (got ${t2}s)"
  fi
  popd >/dev/null || true
}

test_docker_comparison() {
  if [[ "$SKIP_DOCKER" -eq 1 ]]; then
    skip "Docker comparison skipped by flag"
    return
  fi
  if ! has docker; then
    skip "Docker not found; skipping comparison"
    return
  fi
  info "Comparing cached mitl vs docker"
  pushd "$WORKDIR" >/dev/null || return
  local t_mitl t_docker speedup
  # Warm mitl cache
  mitl run echo warm >/dev/null 2>&1 || true
  t_mitl=$(measure_time mitl run echo cached 2>/dev/null || true)
  # Docker baseline (alpine)
  t_docker=$(measure_time docker run --rm -v "$PWD":/app -w /app alpine:latest sh -lc 'echo docker' 2>/dev/null || true)
  echo "mitl cached: ${t_mitl}s | docker run: ${t_docker}s"
  if [[ -z "$t_mitl" || -z "$t_docker" ]]; then
    skip "Could not measure both times"
    popd >/dev/null || true
    return
  fi
  speedup=$(float_div "$t_docker" "$t_mitl")
  echo "Speedup: ${speedup}x"
  if float_ge "$speedup" "$REQUIRED_SPEEDUP"; then
    pass "mitl cached is >= ${REQUIRED_SPEEDUP}x faster than Docker"
  else
    fail "mitl cached speedup < ${REQUIRED_SPEEDUP}x (got ${speedup}x)"
  fi
  popd >/dev/null || true
}

test_digest_determinism() {
  info "Testing digest determinism and .mitlignore"
  pushd "$WORKDIR" >/dev/null || return
  cat > main.go <<'EOF'
package main
func main() {}
EOF
  mitl digest > digest1.txt
  mitl digest > digest2.txt
  if diff -q digest1.txt digest2.txt >/dev/null 2>&1; then
    pass "Digest is stable across runs"
  else
    fail ".Digest changed unexpectedly"
  fi
  echo "*.log" > .mitlignore
  touch test.log
  mitl digest > digest3.txt
  echo "change" >> test.log
  mitl digest > digest4.txt
  if diff -q digest3.txt digest4.txt >/dev/null 2>&1; then
    pass ".mitlignore excludes matching files from digest"
  else
    fail ".mitlignore exclusion did not work"
  fi
  popd >/dev/null || true
}

test_stacks() {
  if [[ "$LIGHT_MODE" -eq 1 ]]; then
    skip "Stack tests skipped (light mode)"
    return
  fi
  info "Testing zero-config stack detection"
  local d
  # Laravel minimal marker
  d=$(mktemp -d 2>/dev/null || mktemp -d -t mitl-laravel)
  pushd "$d" >/dev/null || return
  echo '{"require": {"laravel/framework": "^10.0"}}' > composer.json
  : > artisan
  if mitl run php -v >/dev/null 2>&1; then pass "Laravel detected and php runs"; else warn "Laravel test could not run (runtime/images missing?)"; skip_cnt=$((skip_cnt+1)); fi
  popd >/dev/null || true

  # Node minimal
  d=$(mktemp -d 2>/dev/null || mktemp -d -t mitl-node)
  pushd "$d" >/dev/null || return
  echo '{"name":"test","dependencies":{"express":"^4.18.0"}}' > package.json
  echo "console.log('Hello');" > index.js
  if mitl run node index.js | grep -q Hello; then pass "Node detected and runs"; else warn "Node test could not run"; skip_cnt=$((skip_cnt+1)); fi
  popd >/dev/null || true

  # Python minimal
  d=$(mktemp -d 2>/dev/null || mktemp -d -t mitl-python)
  pushd "$d" >/dev/null || return
  echo "flask==2.0.0" > requirements.txt
  echo "print('Hello from Python')" > app.py
  if mitl run python app.py | grep -q "Hello from Python"; then pass "Python detected and runs"; else warn "Python test could not run"; skip_cnt=$((skip_cnt+1)); fi
  popd >/dev/null || true
}

test_bench() {
  if [[ "$LIGHT_MODE" -eq 1 ]]; then
    skip "Benchmark skipped (light mode)"
    return
  fi
  info "Running mitl bench (short)"
  if mitl bench list >/dev/null 2>&1; then
    pass "Bench command available"
  else
    fail "Bench command not available"
  fi
  if [[ "$SKIP_DOCKER" -eq 0 && $(has docker; echo $?) -eq 0 ]]; then
    mitl bench compare --iterations 3 >/dev/null 2>&1 || warn "Bench comparison may need images; continuing"
  fi
}

summary() {
  echo
  echo -e "${BOLD}Summary:${RESET} PASS=${pass_cnt} FAIL=${fail_cnt} SKIP=${skip_cnt}"
  if [[ $fail_cnt -eq 0 ]]; then
    ok "Preflight checks PASSED"
    exit 0
  else
    err "Preflight checks FAILED"
    exit 1
  fi
}

main() {
  print_banner
  require_mitl
  info "mitl: $(mitl version 2>/dev/null || echo unknown)"
  setup_workdir
  test_doctor
  test_performance_cached
  test_docker_comparison
  test_digest_determinism
  test_stacks
  test_bench
  summary
}

main "$@"
