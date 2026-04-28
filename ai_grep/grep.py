#!/usr/bin/env python3
import sys

if len(sys.argv) < 2:
    sys.stderr.write("usage: grep PATTERN\n")
    sys.exit(1)

pattern = sys.argv[1]
for line in sys.stdin:
    if pattern in line:
        sys.stdout.write(line)
