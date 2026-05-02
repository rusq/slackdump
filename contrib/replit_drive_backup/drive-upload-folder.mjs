/**
 * drive-upload-folder.mjs
 *
 * Recursively uploads a slackdump archive directory to Google Drive,
 * preserving the original folder structure.  Designed to run inside a
 * Replit workspace, using the Replit Google Drive integration so that
 * no OAuth client credentials need to be managed by hand.
 *
 * Strategy:
 *   ALL files use the resumable upload flow, even small ones.  This is
 *   because the Replit connector proxy rejects request bodies larger
 *   than ~1 MB, which makes plain multipart upload unreliable.  With
 *   resumable upload:
 *     1. Initiation (small JSON body) goes through the proxy → returns
 *        a session URL signed by Google.
 *     2. The actual data is PUT directly to Google's session URL,
 *        bypassing the proxy entirely.
 *
 * Features:
 *   - Manifest-based resume: skips already-done files on restart.
 *   - Mid-file resume: re-queries server offset and continues.
 *   - Re-initiates expired resumable sessions (404/410) automatically.
 *   - Exponential backoff on 5xx/429/network errors (up to 5 attempts).
 *   - Periodic progress with throughput and ETA.
 *   - Final reconciliation report comparing local vs Drive byte counts.
 *
 * Usage:
 *   DRIVE_PARENT_ID=<your-folder-id> node drive-upload-folder.mjs
 *
 * Env overrides:
 *   ARCHIVE_DIR          Local archive root (default: ./archive)
 *   DRIVE_PARENT_ID      Drive folder ID to place archive/ under  (REQUIRED)
 *   MANIFEST_PATH        Path to manifest JSON
 *                        (default: ./archive-upload-manifest.json)
 *   REPORT_PATH          Path to report output
 *                        (default: ./archive-upload-report.txt)
 *   PROGRESS_INTERVAL_MS Progress log interval in ms (default: 5000)
 *   CHUNK_SIZE_MB        Chunk size in MB for resumable upload (default: 64)
 *
 * Requirements:
 *   - Node.js 18+ (uses global fetch).
 *   - The "@replit/connectors-sdk" package, installed in this directory.
 *   - The Google Drive integration enabled in your Replit workspace.
 */

import fs from "node:fs";
import path from "node:path";
import { ReplitConnectors } from "@replit/connectors-sdk";

// ─── Config ─────────────────────────────────────────────────────────────────

const ARCHIVE_DIR = process.env.ARCHIVE_DIR
  || path.resolve(process.cwd(), "archive");
const DRIVE_PARENT_ID = process.env.DRIVE_PARENT_ID;
const MANIFEST_PATH = process.env.MANIFEST_PATH
  || path.resolve(process.cwd(), "archive-upload-manifest.json");
const REPORT_PATH = process.env.REPORT_PATH
  || path.resolve(process.cwd(), "archive-upload-report.txt");
const PROGRESS_INTERVAL_MS = parseInt(process.env.PROGRESS_INTERVAL_MS || "5000", 10);
const RESUMABLE_CHUNK_SIZE = parseInt(process.env.CHUNK_SIZE_MB || "64", 10) * 1024 * 1024;
const MAX_RETRIES = 5;

if (!DRIVE_PARENT_ID) {
  console.error("DRIVE_PARENT_ID env var is required (the Drive folder ID to place 'archive/' under).");
  process.exit(2);
}
if (!fs.existsSync(ARCHIVE_DIR)) {
  console.error(`ARCHIVE_DIR does not exist: ${ARCHIVE_DIR}`);
  process.exit(2);
}

const connectors = new ReplitConnectors();

// ─── Utilities ──────────────────────────────────────────────────────────────

function sleep(ms) {
  return new Promise(r => setTimeout(r, ms));
}

function guessMime(filename) {
  const ext = path.extname(filename).toLowerCase();
  const map = {
    ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
    ".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
    ".pdf": "application/pdf",
    ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    ".xls": "application/vnd.ms-excel",
    ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    ".doc": "application/msword",
    ".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    ".ppt": "application/vnd.ms-powerpoint",
    ".zip": "application/zip", ".gz": "application/gzip", ".tar": "application/x-tar",
    ".mp4": "video/mp4", ".mov": "video/quicktime", ".avi": "video/x-msvideo",
    ".mp3": "audio/mpeg", ".wav": "audio/wav",
    ".txt": "text/plain", ".csv": "text/csv", ".html": "text/html",
    ".json": "application/json", ".sqlite": "application/x-sqlite3",
  };
  return map[ext] || "application/octet-stream";
}

// ─── Drive API (via connector proxy) ────────────────────────────────────────
// Only metadata/control calls go through the proxy (small bodies).
// File data is uploaded directly to Google's resumable session URL.

async function proxyRequest(urlPath, options = {}) {
  return connectors.proxy("google-drive", urlPath, options);
}

async function proxyJson(urlPath, options = {}) {
  const res = await proxyRequest(urlPath, options);
  const text = await res.text();
  if (!res.ok) {
    const err = Object.assign(
      new Error(`Drive API ${res.status}: ${text.slice(0, 400)}`),
      { status: res.status, body: text },
    );
    throw err;
  }
  return JSON.parse(text);
}

async function withRetry(label, fn) {
  let lastErr;
  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    try {
      return await fn(attempt);
    } catch (err) {
      const status = err.status;
      const retryable = !status || status === 429 || status >= 500;
      if (!retryable) throw err;
      lastErr = err;
      const delay = Math.min(2000 * Math.pow(2, attempt - 1), 60000);
      console.error(`  [retry ${attempt}/${MAX_RETRIES}] ${label}: ${String(err.message).slice(0, 100)} — wait ${delay}ms`);
      await sleep(delay);
    }
  }
  throw lastErr;
}

// ─── Folder management ──────────────────────────────────────────────────────

const folderCache = new Map();

async function ensureFolder(name, parentId) {
  const key = `${parentId}/${name}`;
  if (folderCache.has(key)) return folderCache.get(key);

  const q = encodeURIComponent(
    `name='${name.replace(/\\/g, "\\\\").replace(/'/g, "\\'")}' and mimeType='application/vnd.google-apps.folder' and '${parentId}' in parents and trashed=false`,
  );
  const list = await withRetry(`list folder "${name}"`, () =>
    proxyJson(`/drive/v3/files?q=${q}&fields=files(id,name)`),
  );
  if (list.files?.length > 0) {
    folderCache.set(key, list.files[0].id);
    return list.files[0].id;
  }

  const created = await withRetry(`create folder "${name}"`, () =>
    proxyJson("/drive/v3/files", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, mimeType: "application/vnd.google-apps.folder", parents: [parentId] }),
    }),
  );
  folderCache.set(key, created.id);
  return created.id;
}

async function ensureFolderPath(relDir, rootId) {
  if (!relDir) return rootId;
  const parts = relDir.split("/").filter(Boolean);
  let cur = rootId;
  for (const p of parts) cur = await ensureFolder(p, cur);
  return cur;
}

// ─── Resumable upload ───────────────────────────────────────────────────────

async function initiateResumable(filename, fileSize, mimeType, parentFolderId) {
  const metadata = JSON.stringify({ name: filename, parents: [parentFolderId] });
  return withRetry(`initiate session "${filename}"`, async () => {
    const res = await proxyRequest(
      "/upload/drive/v3/files?uploadType=resumable&fields=id,name,size",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json; charset=UTF-8",
          "X-Upload-Content-Type": mimeType,
          "X-Upload-Content-Length": String(fileSize),
        },
        body: metadata,
      },
    );
    const text = await res.text();
    if (!res.ok) {
      throw Object.assign(
        new Error(`Initiate ${res.status}: ${text.slice(0, 300)}`),
        { status: res.status },
      );
    }
    const location = res.headers.get("location") || res.headers.get("Location");
    if (location) return location;
    try { const j = JSON.parse(text); if (j.uploadUrl) return j.uploadUrl; } catch (_) { /* not JSON */ }
    throw new Error(`No Location in initiation response. Body: ${text.slice(0, 200)}`);
  });
}

async function queryOffset(uploadUrl, fileSize) {
  const res = await fetch(uploadUrl, {
    method: "PUT",
    headers: { "Content-Range": `bytes */${fileSize}` },
  });
  if (res.status === 200 || res.status === 201) {
    const j = await res.json();
    return { complete: true, fileId: j.id, name: j.name };
  }
  if (res.status === 308) {
    const range = res.headers.get("range");
    const m = range?.match(/bytes=0-(\d+)/);
    return { complete: false, offset: m ? parseInt(m[1]) + 1 : 0 };
  }
  const text = await res.text();
  throw Object.assign(
    new Error(`queryOffset ${res.status}: ${text.slice(0, 200)}`),
    { status: res.status },
  );
}

async function putChunk(uploadUrl, buf, offset, fileSize) {
  const end = offset + buf.length - 1;
  const res = await fetch(uploadUrl, {
    method: "PUT",
    headers: {
      "Content-Length": String(buf.length),
      "Content-Range": `bytes ${offset}-${end}/${fileSize}`,
    },
    body: buf,
  });
  if (res.status === 200 || res.status === 201) {
    const j = await res.json();
    return { done: true, fileId: j.id, name: j.name };
  }
  if (res.status === 308) {
    const range = res.headers.get("range");
    const m = range?.match(/bytes=0-(\d+)/);
    return { done: false, nextOffset: m ? parseInt(m[1]) + 1 : offset + buf.length };
  }
  const text = await res.text();
  throw Object.assign(
    new Error(`putChunk ${res.status}: ${text.slice(0, 300)}`),
    { status: res.status },
  );
}

async function uploadFile(localPath, filename, fileSize, parentFolderId, manifestEntry, saveManifest) {
  const mimeType = guessMime(filename);
  let uploadUrl = manifestEntry.uploadUrl || null;
  let offset = manifestEntry.offset || 0;

  if (uploadUrl && offset > 0) {
    try {
      const q = await queryOffset(uploadUrl, fileSize);
      if (q.complete) return { fileId: q.fileId, name: q.name };
      offset = q.offset;
    } catch (err) {
      if (err.status === 404 || err.status === 410) {
        console.log(`  Session expired, re-initiating for "${filename}"`);
        uploadUrl = null;
        offset = 0;
      } else {
        throw err;
      }
    }
  }

  if (!uploadUrl) {
    uploadUrl = await initiateResumable(filename, fileSize, mimeType, parentFolderId);
    offset = 0;
    manifestEntry.uploadUrl = uploadUrl;
    manifestEntry.offset = 0;
    saveManifest();
  }

  const fd = fs.openSync(localPath, "r");
  try {
    while (offset < fileSize) {
      const chunkLen = Math.min(RESUMABLE_CHUNK_SIZE, fileSize - offset);
      const buf = Buffer.alloc(chunkLen);
      fs.readSync(fd, buf, 0, chunkLen, offset);

      let result;
      for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
        try {
          result = await putChunk(uploadUrl, buf, offset, fileSize);
          break;
        } catch (err) {
          if (err.status === 404 || err.status === 410) {
            console.log(`  Session expired mid-upload, re-initiating for "${filename}"`);
            uploadUrl = await initiateResumable(filename, fileSize, mimeType, parentFolderId);
            offset = 0;
            manifestEntry.uploadUrl = uploadUrl;
            manifestEntry.offset = 0;
            saveManifest();
            result = null;
            break;
          }
          const retryable = !err.status || err.status === 429 || err.status >= 500;
          if (!retryable || attempt >= MAX_RETRIES) throw err;
          const delay = Math.min(2000 * Math.pow(2, attempt - 1), 60000);
          console.error(`  [retry ${attempt}/${MAX_RETRIES}] chunk@${offset} "${filename}": ${err.message.slice(0, 80)} — ${delay}ms`);
          await sleep(delay);
        }
      }

      if (!result) continue;
      if (result.done) return { fileId: result.fileId, name: result.name };

      offset = result.nextOffset;
      manifestEntry.offset = offset;
      saveManifest();
    }
  } finally {
    fs.closeSync(fd);
  }

  throw new Error(`Upload loop ended without completion for "${filename}"`);
}

// ─── Manifest ───────────────────────────────────────────────────────────────

function loadManifest() {
  if (fs.existsSync(MANIFEST_PATH)) {
    try { return JSON.parse(fs.readFileSync(MANIFEST_PATH, "utf8")); } catch (_) { /* corrupt; start fresh */ }
  }
  return {};
}

function saveManifest(manifest) {
  fs.writeFileSync(MANIFEST_PATH, JSON.stringify(manifest, null, 2));
}

// ─── Drive file count (for reconciliation) ──────────────────────────────────

async function countDriveFolder(folderId) {
  let fileCount = 0;
  let byteCount = 0;
  const queue = [folderId];
  while (queue.length > 0) {
    const id = queue.shift();
    let pageToken = null;
    do {
      const q = encodeURIComponent(`'${id}' in parents and trashed=false`);
      let url = `/drive/v3/files?q=${q}&fields=files(id,mimeType,size),nextPageToken&pageSize=1000`;
      if (pageToken) url += `&pageToken=${encodeURIComponent(pageToken)}`;
      let list;
      try {
        list = await withRetry("list drive files", () => proxyJson(url));
      } catch (err) {
        console.error(`  [WARN] Cannot list Drive folder ${id}: ${err.message}`);
        break;
      }
      for (const f of list.files || []) {
        if (f.mimeType === "application/vnd.google-apps.folder") {
          queue.push(f.id);
        } else {
          fileCount++;
          byteCount += parseInt(f.size || "0", 10);
        }
      }
      pageToken = list.nextPageToken;
    } while (pageToken);
  }
  return { fileCount, byteCount };
}

// ─── Main ───────────────────────────────────────────────────────────────────

async function main() {
  console.log("=== Slackdump archive → Google Drive uploader ===");
  console.log(`Archive     : ${ARCHIVE_DIR}`);
  console.log(`Drive parent: ${DRIVE_PARENT_ID}`);
  console.log(`Manifest    : ${MANIFEST_PATH}`);
  console.log(`Chunk size  : ${RESUMABLE_CHUNK_SIZE / 1024 / 1024} MB`);
  console.log();

  console.log("Scanning local archive...");
  const allFiles = [];
  function walk(dir, relBase) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      const rel = relBase ? `${relBase}/${entry.name}` : entry.name;
      if (entry.isDirectory()) walk(full, rel);
      else if (entry.isFile()) {
        const st = fs.statSync(full);
        allFiles.push({ fullPath: full, relPath: rel, size: st.size, mtimeMs: st.mtimeMs });
      }
    }
  }
  walk(ARCHIVE_DIR, "");
  const totalFiles = allFiles.length;
  const totalBytes = allFiles.reduce((s, f) => s + f.size, 0);
  console.log(`Found ${totalFiles} files, ${(totalBytes / 1024 / 1024 / 1024).toFixed(2)} GB`);
  console.log();

  console.log("Ensuring 'archive' folder in Drive...");
  const archiveFolderId = await ensureFolder("archive", DRIVE_PARENT_ID);
  console.log(`archive folder id: ${archiveFolderId}`);
  console.log();

  const manifest = loadManifest();
  const saveNow = () => saveManifest(manifest);

  let doneCount = 0;
  let doneBytes = 0;
  for (const f of allFiles) {
    const e = manifest[f.relPath];
    if (e?.status === "done") {
      const sizeMatch = e.size == null || e.size === f.size;
      const mtimeMatch = e.mtimeMs == null || e.mtimeMs === f.mtimeMs;
      if (!sizeMatch || !mtimeMatch) {
        console.log(`  [STALE] ${f.relPath}: manifest size/mtime mismatch — will re-upload`);
        manifest[f.relPath] = { status: "pending" };
        continue;
      }
      doneCount++;
      doneBytes += f.size;
    }
  }
  console.log(`Resuming: ${doneCount}/${totalFiles} files already done (${(doneBytes / 1024 / 1024).toFixed(0)} MB)`);
  console.log();

  let processedCount = doneCount;
  let processedBytes = doneBytes;
  let failedCount = 0;
  const failed = [];
  let lastProgress = Date.now();
  const sessionStartTime = Date.now();
  const sessionStartBytes = doneBytes;

  function printProgress() {
    const pct = totalFiles > 0 ? (processedCount / totalFiles * 100).toFixed(1) : "0.0";
    const gbDone = (processedBytes / 1024 / 1024 / 1024).toFixed(2);
    const gbTotal = (totalBytes / 1024 / 1024 / 1024).toFixed(2);
    const elapsedSec = (Date.now() - sessionStartTime) / 1000;
    const sessionBytes = processedBytes - sessionStartBytes;
    const throughputMBps = elapsedSec > 0 ? (sessionBytes / 1024 / 1024 / elapsedSec) : 0;
    const remainingBytes = totalBytes - processedBytes;
    const etaSec = throughputMBps > 0 ? remainingBytes / 1024 / 1024 / throughputMBps : Infinity;
    const etaStr = !isFinite(etaSec) ? "-" : etaSec < 60
      ? `${Math.round(etaSec)}s`
      : etaSec < 3600
        ? `${Math.round(etaSec / 60)}m`
        : `${(etaSec / 3600).toFixed(1)}h`;
    console.log(
      `Progress: ${processedCount}/${totalFiles} (${pct}%) | ${gbDone}/${gbTotal} GB`
      + ` | ${throughputMBps.toFixed(2)} MB/s | ETA: ${etaStr} | failed: ${failedCount}`,
    );
  }

  for (const { fullPath, relPath, size, mtimeMs } of allFiles) {
    if (manifest[relPath]?.status === "done") continue;

    const now = Date.now();
    if (now - lastProgress >= PROGRESS_INTERVAL_MS) {
      printProgress();
      lastProgress = now;
    }

    const relDir = path.dirname(relPath);
    const filename = path.basename(relPath);
    let parentFolderId;
    try {
      parentFolderId = await ensureFolderPath(relDir === "." ? "" : relDir, archiveFolderId);
    } catch (err) {
      console.error(`  [ERROR] folder for "${relPath}": ${err.message}`);
      manifest[relPath] = { status: "failed", error: err.message };
      saveNow();
      failedCount++;
      failed.push({ relPath, error: err.message });
      processedCount++;
      processedBytes += size;
      continue;
    }

    if (!manifest[relPath]) manifest[relPath] = { status: "pending" };
    manifest[relPath].status = "uploading";

    try {
      const { fileId, name } = await uploadFile(
        fullPath, filename, size, parentFolderId,
        manifest[relPath], saveNow,
      );
      manifest[relPath] = { status: "done", fileId, name, size, mtimeMs };
      saveNow();
      processedCount++;
      processedBytes += size;
    } catch (err) {
      const msg = String(err.message).slice(0, 300);
      console.error(`  [FAILED] ${relPath}: ${msg}`);
      manifest[relPath] = {
        status: "failed",
        error: msg,
        ...(manifest[relPath].uploadUrl ? { uploadUrl: manifest[relPath].uploadUrl } : {}),
      };
      saveNow();
      failedCount++;
      failed.push({ relPath, error: msg });
      processedCount++;
      processedBytes += size;
    }
  }

  printProgress();

  console.log();
  console.log("Counting files on Drive...");
  const { fileCount: driveFileCount, byteCount: driveBytes } = await countDriveFolder(archiveFolderId);

  const doneInManifest = Object.values(manifest).filter(e => e.status === "done").length;
  const report = [
    "=== archive Upload Reconciliation Report ===",
    `Date: ${new Date().toISOString()}`,
    "",
    `Local files scanned  : ${totalFiles}`,
    `Local total bytes    : ${totalBytes} (${(totalBytes / 1024 / 1024 / 1024).toFixed(2)} GB)`,
    "",
    `Manifest done        : ${doneInManifest}`,
    `Drive files counted  : ${driveFileCount}`,
    `Drive total bytes    : ${driveBytes} (${(driveBytes / 1024 / 1024 / 1024).toFixed(2)} GB)`,
    "",
    `Failed uploads       : ${failed.length}`,
    ...(failed.length > 0 ? ["", "Failed files:"] : []),
    ...failed.map(f => `  ${f.relPath}: ${f.error}`),
    "",
    `Byte match  : ${totalBytes === driveBytes ? "YES" : `NO - local ${totalBytes} vs drive ${driveBytes}`}`,
    `Count match : ${totalFiles === driveFileCount ? "YES" : `NO - local ${totalFiles} vs drive ${driveFileCount}`}`,
  ].join("\n");

  fs.writeFileSync(REPORT_PATH, report);
  console.log("\n" + report);
  console.log(`\nReport: ${REPORT_PATH}`);

  if (failed.length > 0) {
    console.log(`\n${failed.length} file(s) failed. Re-run to retry.`);
    process.exit(1);
  }
  console.log("\nAll files uploaded successfully.");
}

main().catch(err => {
  console.error("Fatal:", err);
  process.exit(1);
});
