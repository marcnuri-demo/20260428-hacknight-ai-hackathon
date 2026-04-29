#!/usr/bin/env bash
# Validates one of the README pipelines by running the native Unix pipeline
# and the AI container pipeline against server.log and diffing the outputs.
# Exits 0 only when the diff is empty.
#
# Usage: validate.sh <problem-number>

set -euo pipefail

problem="${1:-}"

if [[ -z "$problem" ]]; then
  echo "Usage: $0 <problem-number>" >&2
  exit 2
fi

case "$problem" in
  0)
    native=$(cat server.log \
      | grep "ERROR" \
      | wc -l)

    ai=$(cat server.log \
      | docker run --rm -i ai_grep "ERROR" \
      | docker run --rm -i ai_wc -l)
    ;;
  1)
    native=$(cat server.log \
      | grep "ERROR" \
      | grep -oE "[A-Za-z ]+$" \
      | sort \
      | uniq -c \
      | sort -rn)

    ai=$(cat server.log \
      | docker run --rm -i ai_grep "ERROR" \
      | docker run --rm -i ai_grep -oE "[A-Za-z ]+$" \
      | docker run --rm -i ai_sort \
      | docker run --rm -i ai_uniq -c \
      | docker run --rm -i ai_sort -rn)
    ;;
  2)
    native=$(cat server.log \
      | grep "ERROR" \
      | sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
      | sort -u \
      | sed 's/|.*//' \
      | sort \
      | uniq -c \
      | sort -rn)

    ai=$(cat server.log \
      | docker run --rm -i ai_grep "ERROR" \
      | docker run --rm -i ai_sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
      | docker run --rm -i ai_sort -u \
      | docker run --rm -i ai_sed 's/|.*//' \
      | docker run --rm -i ai_sort \
      | docker run --rm -i ai_uniq -c \
      | docker run --rm -i ai_sort -rn)
    ;;
  3)
    native=$(cat server.log \
      | grep "ERROR" \
      | grep -E "^\[2026-04-16 (0[0-8]|1[7-9]|2[0-3]):" \
      | sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
      | sort -u \
      | sed 's/|.*//' \
      | sort \
      | uniq -c \
      | sort -rn)

    ai=$(cat server.log \
      | docker run --rm -i ai_grep "ERROR" \
      | docker run --rm -i ai_grep -E "^\[2026-04-16 (0[0-8]|1[7-9]|2[0-3]):" \
      | docker run --rm -i ai_sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
      | docker run --rm -i ai_sort -u \
      | docker run --rm -i ai_sed 's/|.*//' \
      | docker run --rm -i ai_sort \
      | docker run --rm -i ai_uniq -c \
      | docker run --rm -i ai_sort -rn)
    ;;
  *)
    echo "Unknown problem: $problem" >&2
    exit 2
    ;;
esac

# `diff -w` ignores whitespace differences between the native and AI
# pipelines. BSD `wc -l` (and `uniq -c`) right-pad numeric output with
# leading spaces; GNU equivalents do not. Both outputs are functionally
# identical — same counts, same ordering — so we treat whitespace-only
# differences as equivalent rather than forcing every AI tool to mimic
# the host's platform-specific padding.
if diff -w <(echo "$native") <(echo "$ai"); then
  echo "Problem $problem: PASS"
else
  echo "Problem $problem: FAIL" >&2
  exit 1
fi
