#!/usr/bin/env python
import io
import json
import argparse

RECORD_FILE = "record.jsonl"
INDEX_FILE = "index.json"


def index_stats(f: io.TextIOWrapper):
    index = json.load(f)

    entries = 0
    channels = 0
    threads = 0
    files = 0
    for k, v in index.items():
        print("Key: %s, value: %s" % (k, v))
        entries += len(v)
        if k.startswith("t"):
            threads += 1
        elif k.startswith("f"):
            files += 1
        else:
            channels += 1

    print("Total number of index entries: %d" % len(index))
    print("Total number of data offsets: %d" % entries)
    print("Total number of channels: %d" % channels)
    print("Total number of threads: %d" % threads)
    print("Total number of files: %d" % files)


def record_stats(f: io.TextIOWrapper):
    messages = 0
    msg_requests = 0
    threads = 0
    thread_requests = 0
    files = 0
    file_requests = 0
    total = 0
    for line in f:
        total += 1
        data = json.loads(line)
        if data["type"] == 0:
            msg_requests += 1
            messages += data["size"]
        elif data["type"] == 1:
            thread_requests += 1
            threads += data["size"]
        elif data["type"] == 2:
            file_requests += 1
            files += data["size"]
    print("Total number of message requests: {}, messages {}".format(
        msg_requests, messages))
    print("Total number of thread requests: {}, thread messages: {}".format(
        thread_requests, threads))
    print("Total number of file requests: {}, files: {}".format(file_requests, files))
    print("Total number of API requests: {}".format(total))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Record file statistics tool")
    parser.add_argument("file", nargs="?", default=RECORD_FILE,
                        help="Record or index file to analyze (default: record.jsonl)")
    parser.add_argument("-i", "--index", action="store_true", default=False,
                        help="Treat the file as index file if true, otherwise treat it as record file.")
    args = parser.parse_args()

    if args.index:
        fn = index_stats
    else:
        fn = record_stats

    try:
        with open(args.file) as f:
            fn(f)
    except FileNotFoundError:
        print("File not found: {}".format(args.file))
