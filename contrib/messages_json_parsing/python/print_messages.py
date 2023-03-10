#!/usr/bin/env python
import os
import json

SAMPLE = os.path.join("..", "..", "sample.json")


def print_messages(filename: str) -> None:
    """print_messages prints all messages from json file ``filename``"""
    if not filename:
        raise RuntimeError("filename is empty")

    f = open(filename)
    conversation = json.load(f)
    for msg in conversation['messages']:
        print_msg(msg)
        if replies := msg.get('slackdump_thread_replies'):
            for reply in replies:
                print_msg(reply, "...")


def print_msg(m: dict, indent: str = "") -> None:
    """print_message prints the message ``m``, indenting the line with
    ``indent``"""
    print(f"{indent}{m['user']}: {m['text']}")


if __name__ == "__main__":
    print_messages(SAMPLE)
