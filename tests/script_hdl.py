#!/usr/bin/env python3

from sys import argv, stderr

if __name__ == "__main__":
    stderr.write(f"SCRIPT_HDL {' '.join(argv)}\n")
    stderr.flush()
