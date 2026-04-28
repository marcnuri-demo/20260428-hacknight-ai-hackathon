# AI Hacknight

You're at a hacknight. The challenge: use AI (Claude, Copilot, whatever) to implement classic Unix tools from scratch — then prove they actually work.

The theme running through the problems is processing a web server log. You'll answer questions like "how many errors happened?", "which error types are most common?", and "which users were affected?" — all using Unix command line tools like `grep`, `sort`, `uniq`, `sed`, and `wc` piped together.

The twist is you already have the answer. The native Unix tools are your ground truth: feed both pipelines the same `server.log` and compare. If your AI-built versions match, you win.

The goal is to implement the full AI pipeline for each problem, but you don't have to do it all at once. You can implement one tool at a time and mix native and AI versions in the validate step. For example, if you've only implemented `ai_grep`, you can verify it in isolation by keeping the rest of the pipeline native:

```bash
native=$(cat server.log \
  | grep "ERROR" \
  | wc -l)

ai=$(cat server.log \
  | docker run --rm -i ai_grep "ERROR" \
  | wc -l)

diff <(echo "$native") <(echo "$ai")
```

Once that passes, move on to the next tool.

## Requirements

- macOS or Linux
- Docker
- bash (not sh or fish — validate commands use process substitution)

## Problem 0

How many errors are in `server.log`?

This problem is already solved — use it as a reference for the problems ahead.

The native Unix pipeline answers it in one line:
```bash
cat server.log \
  | grep "ERROR" \
  | wc -l
```

The AI-generated equivalent uses the same pipeline, but each tool is a Docker image built from scratch. You'll find the implementations in `ai_grep/` and `ai_wc/`.

Build the images:
```bash
docker build -t ai_grep ai_grep \
  && docker build -t ai_wc ai_wc
```

Run the AI pipeline:
```bash
cat server.log \
  | docker run --rm -i ai_grep "ERROR" \
  | docker run --rm -i ai_wc -l
```

Validate both produce the same result:
```bash
native=$(cat server.log \
  | grep "ERROR" \
  | wc -l)

ai=$(cat server.log \
  | docker run --rm -i ai_grep "ERROR" \
  | docker run --rm -i ai_wc -l)

diff <(echo "$native") <(echo "$ai")
```

No output means the results match.

## Problem 1

Which error types appear in `server.log`, and how many times each?

The native Unix pipeline:
```bash
cat server.log \
  | grep "ERROR" \
  | grep -oE "[A-Za-z ]+$" \
  | sort \
  | uniq -c \
  | sort -rn
```

Build the images (same as Problem 0):
```bash
docker build -t ai_grep ai_grep \
  && docker build -t ai_sort ai_sort \
  && docker build -t ai_uniq ai_uniq
```

Run the AI pipeline:
```bash
cat server.log \
  | docker run --rm -i ai_grep "ERROR" \
  | docker run --rm -i ai_grep -oE "[A-Za-z ]+$" \
  | docker run --rm -i ai_sort \
  | docker run --rm -i ai_uniq -c \
  | docker run --rm -i ai_sort -rn
```

Validate both produce the same result:
```bash
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

diff <(echo "$native") <(echo "$ai")
```

No output means the results match.

## Problem 2

How many unique users were affected by each error type?

The native Unix pipeline:
```bash
cat server.log \
  | grep "ERROR" \
  | sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
  | sort -u \
  | sed 's/|.*//' \
  | sort \
  | uniq -c \
  | sort -rn
```

Build the images:
```bash
docker build -t ai_grep ai_grep \
  && docker build -t ai_sed ai_sed \
  && docker build -t ai_sort ai_sort \
  && docker build -t ai_uniq ai_uniq
```

Run the AI pipeline:
```bash
cat server.log \
  | docker run --rm -i ai_grep "ERROR" \
  | docker run --rm -i ai_sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
  | docker run --rm -i ai_sort -u \
  | docker run --rm -i ai_sed 's/|.*//' \
  | docker run --rm -i ai_sort \
  | docker run --rm -i ai_uniq -c \
  | docker run --rm -i ai_sort -rn
```

Validate both produce the same result:
```bash
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

diff <(echo "$native") <(echo "$ai")
```

No output means the results match.

## Problem 3

How many unique users were affected by each error type **outside working hours** (before 09:00 or from 17:00 onwards)?

The native Unix pipeline:
```bash
cat server.log \
  | grep "ERROR" \
  | grep -E "^\[2026-04-16 (0[0-8]|1[7-9]|2[0-3]):" \
  | sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
  | sort -u \
  | sed 's/|.*//' \
  | sort \
  | uniq -c \
  | sort -rn
```

Build the images:
```bash
docker build -t ai_grep ai_grep \
  && docker build -t ai_sed ai_sed \
  && docker build -t ai_sort ai_sort \
  && docker build -t ai_uniq ai_uniq
```

Run the AI pipeline:
```bash
cat server.log \
  | docker run --rm -i ai_grep "ERROR" \
  | docker run --rm -i ai_grep -E "^\[2026-04-16 (0[0-8]|1[7-9]|2[0-3]):" \
  | docker run --rm -i ai_sed 's/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/' \
  | docker run --rm -i ai_sort -u \
  | docker run --rm -i ai_sed 's/|.*//' \
  | docker run --rm -i ai_sort \
  | docker run --rm -i ai_uniq -c \
  | docker run --rm -i ai_sort -rn
```

Validate both produce the same result:
```bash
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

diff <(echo "$native") <(echo "$ai")
```

No output means the results match.

## Problem 4

Clone & Verify: pick any Unix tool (`grep`, `sort`, `uniq`, `wc`, or `sed`) and build a true behavioral clone — one that is indistinguishable from the real thing for any valid input and any supported flag.

Start by reading the man page to understand the full interface:

```bash
man grep   # or sort, uniq, wc, sed
```

Then build a test suite that runs the same inputs through both the native tool and your Docker image and asserts the outputs are identical. The test suite is the proof — it should be runnable by anyone and cover enough cases to make a convincing argument.

There is no prescribed format: a shell script, a Python test file, a table of cases — whatever makes the evidence clear and executable.
