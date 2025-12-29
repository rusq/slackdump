#!/usr/bin/env python3
import os
import json
import re

PORT = 8000
BASE_IP = "127.0.0.1"
BASE_URL = f"http://{BASE_IP}:{PORT}"

KEYS_TO_UPDATE = [
    "url_private", "thumb_64", "url_private_download", "thumb_80",
    "thumb_160", "thumb_360", "thumb_360_gif", "permalink",
    "permalink_public", "thumb_480", "thumb_720", "thumb_960", "thumb_1024",
]

def process_json(data, attachments_map, stats):
    if isinstance(data, dict):
        # Check if this is a file object with a name field
        if "name" in data and "url_private" in data:
            filename = data["name"]
            if filename in attachments_map:
                stats["files_found"] += 1
                new_url = f"{BASE_URL}/{attachments_map[filename]}"
                for k in KEYS_TO_UPDATE:
                    if k in data and data[k]:  # Only process non-empty URLs
                        old_url = data[k]
                        data[k] = new_url
                        stats["urls_replaced"] += 1
            else:
                stats["files_not_found"] += 1
                stats["not_found_files"].add(filename)
        else:
            # Recursively process nested objects
            for key, value in data.items():
                if isinstance(value, (dict, list)):
                    process_json(value, attachments_map, stats)
    elif isinstance(data, list):
        for item in data:
            process_json(item, attachments_map, stats)

def rewrite_all_jsons():
    cwd = os.getcwd()
    attachments_map = {}
    stats = {
        "files_found": 0,
        "files_not_found": 0,
        "urls_replaced": 0,
        "not_found_files": set()
    }

    # Build mapping of original filenames to their actual paths (including subdirectories)
    for root, dirs, files in os.walk(cwd):
        for fname in files:
            if "-" in fname and os.path.splitext(fname)[1]:
                original_name = fname.split("-", 1)[1]
                safe_name = re.sub(r"[^a-zA-Z0-9._-]", "_", original_name)
                # Get relative path from cwd
                rel_path = os.path.relpath(os.path.join(root, fname), cwd)
                attachments_map[original_name] = rel_path
                attachments_map[safe_name] = rel_path
    for root, dirs, files in os.walk(cwd):
        for fname in files:
            if not fname.endswith(".json"):
                continue
            fpath = os.path.join(root, fname)
            with open(fpath, "r", encoding="utf-8") as f:
                try:
                    data = json.load(f)
                except json.JSONDecodeError:
                    continue
            process_json(data, attachments_map, stats)
            with open(fpath, "w", encoding="utf-8") as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
            print(f"Updated {os.path.relpath(fpath, cwd)}")

    # Print summary
    print(f"\n📊 SUMMARY:")
    print(f"Files found and processed: {stats['files_found']}")
    print(f"Files not found: {stats['files_not_found']}")
    print(f"Total URLs replaced: {stats['urls_replaced']}")
    
    if stats['not_found_files']:
        print(f"\n❌ Files not found ({len(stats['not_found_files'])}):")
        for filename in sorted(stats['not_found_files']):
            print(f"  - {filename}")
    
    print(f"\n✅ All Slackdump URLs now point to {BASE_URL}")
    print("Instructions for Windows machine:")
    print("1. Copy this entire folder (JSON + attachments) to the Windows machine running Slackord2")
    print("2. Open Command Prompt and cd into the folder")
    print(f"3. Run: python -m http.server {PORT}")
    print("4. Keep the server running while Slackord2 imports channels")

if __name__ == "__main__":
    rewrite_all_jsons()
