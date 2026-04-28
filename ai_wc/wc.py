#!/usr/bin/env python3
import sys

lines = 0
for line in sys.stdin:
    lines += 1

print(f"{lines:>8}")
