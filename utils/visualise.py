#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Visualises the chunks.

Usage: visualise.py <file>

Example:
    python3 visualise.py ../data/2021-01-01.jsonl | dot -Tpng -o graph.png

It will generate a graph of the messages, threads and files.  Files, that have
an underscore prefix, are files that are attached to a message.  Files, that
have no prefix, are files that are in the files chunk, so, if each message
that has file attachments must have two nodes for each file linked to it.
"""
import sys
import json

CHUNK_MESSAGE = 0
CHUNK_THREAD = 1
CHUNK_FILE = 2

COLOR_MSG= "#54AEA6"
COLOR_MSG_FILE = "#00FFFF"
COLOR_THREAD = "#E0CA87"
COLOR_FILE = "#C4B7D5"

def main(args: list[str]):
    """
    Main function
    """
    if len(args) != 1:
        print("Usage: visualise.py <file>")
        print("Example: python3 visualise.py ../data/2021-01-01.jsonl | dot -Tpng -o graph.png")
        sys.exit(1)

    with open(args[0], "r") as file:
        print("digraph {")
        print("rankdir=LR;")
        print("node [shape=box];")
        for line in file:
            chunk = json.loads(line)
            chunk_type = chunk["t"]
            if chunk_type == CHUNK_MESSAGE:
                for msg in chunk["m"]:
                    print(f"{msg['ts']} [fillcolor=\"{COLOR_MSG}\"; style=filled];")
                    if files := msg.get("files"):
                        if files:
                            for file in files:
                                print(f"_{file['id']}[fillcolor=\"{COLOR_MSG_FILE}\"; style=filled];")
                                print(f"{msg['ts']} -> _{file['id']};")
            elif chunk_type == CHUNK_THREAD:
                for msg in chunk["m"]:
                    print(f"{msg['ts']}[fillcolor=\"{COLOR_THREAD}\"; style=filled];")
                    print(f"{chunk['p']['ts']} -> {msg['ts']};")
                    if files := msg.get("files"):
                        if files:
                            for file in files:
                                print(f"_{file['id']}[fillcolor=\"{COLOR_MSG_FILE}\"; style=filled];")
                                print(f"{msg['ts']} -> _{file['id']};")
            elif chunk_type == CHUNK_FILE:
                for file in chunk["f"]:
                    print(f"{file['id']}[fillcolor=\"{COLOR_FILE}\"; style=filled];")
                    print(f"{chunk['_p']['ts']} -> {file['id']};")
            else:
                # raise ValueError("Unknown chunk type: " + str(chunk_type))
                pass
        print("}")
if __name__ == '__main__':
    main(sys.argv[1:])
