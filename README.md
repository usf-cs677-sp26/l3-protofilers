## Changes from Initial Implementation

### Fixes

- **Fixed server deadlock on PUT**: `handleStorage()` was missing a `SendResponse()` after checksum verification, causing the client to block forever on `ReceiveResponse()`. Added success/failure response and cleanup of corrupt files on checksum mismatch.
- **Fixed client GET ignoring destination directory**: The `dir` argument was parsed but never passed to `get()`. Files now correctly go to the specified directory using `filepath.Join(dir, fileName)`. Also reordered file creation to happen after the server confirms the file exists, avoiding empty leftover files on error.
- **Fixed server crash on missing file retrieval**: `handleRetrieval()` used `log.Fatalln()` which killed the entire server process. Replaced with a graceful error response to the client so the server stays up.

### Performance Improvements (for large files, 100+ GB)

- **Buffered I/O**: Added 1 MB `bufio.Reader`/`bufio.Writer` to `MessageHandler`, reducing syscalls from millions to ~100K for a 100 GB transfer.
- **Larger copy buffer**: Replaced `io.CopyN` (32 KB default) with `util.CopyN` using a 1 MB buffer via `io.CopyBuffer` + `io.LimitReader`, reducing copy-loop iterations ~32x.
- **Disk space checks**: Server checks available space before accepting a PUT; client checks before creating the output file on GET. Uses `syscall.Statfs`.

### Code Refactoring

- **`util.CopyWithChecksum(dst, src, n)`**: Extracted the repeated pattern of creating an MD5 hash, wrapping with `io.MultiWriter`, allocating a 1 MB buffer, and copying — used in all four transfer paths (`put`, `get`, `handleStorage`, `handleRetrieval`).
- **`util.HasDiskSpace(path, needed)`**: Extracted the duplicated `syscall.Statfs` disk space check from both server and client into a shared helper.
- **`msgHandler.ReceiveChecksum()`**: Added to `MessageHandler` following the same pattern as `ReceiveResponse()` and `ReceiveRetrievalResponse()`, replacing the repeated `Receive()` + `.GetChecksum().Checksum` extraction.

### Messaging Improvements

- **Server storage responses** include transfer time (e.g., `"Storage complete: checksum verified, time: 1.23s"`).
- **Server retrieval responses** include file name and size before streaming, and transfer time after streaming completes.
- **Client PUT** prints file name, size, and checksum after transfer.
- **Client GET** prints file name and size when the download begins.
