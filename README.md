# file-transfer

A file transfer client and server using TCP and Google Protocol Buffers.

## Usage

### Server
```bash
./server listen-port [download-dir]
# Example:
./server 9898 ./stuff/
```
If `download-dir` is not provided, files are stored in the current working directory.

### Client
```bash
./client host:port put|get file-name [destination-dir]
# Store a file:
./client localhost:9898 put /some/file.jpg
# Retrieve a file:
./client localhost:9898 get file.jpg /tmp/my/stuff/
```
If `destination-dir` is not provided for GET, files are saved to the current working directory.

## Changes Made

- **Server PUT response**: Added success/failure response to the client after checksum verification so the client no longer hangs
- **Server GET error handling**: Replaced `log.Fatalln` with an error response to the client so the server doesn't crash when a file is not found
- **Server disk space check**: Added a check using `syscall.Statfs` to verify there is enough disk space before accepting a file
- **Client PUT base filename**: Client now strips the path and sends only the base filename to the server
- **Client GET destination directory**: Client now uses the destination directory argument when saving retrieved files
- **Client GET ordering**: Moved file creation to after receiving a successful response from the server
- **Proto StorageRequest checksum**: Added checksum field to `StorageRequest` so the client sends the checksum with the initial request before streaming
- **Proto RetrievalResponse checksum**: Added checksum field to `RetrievalResponse` so the server sends the checksum with the file metadata before streaming
- **Removed separate ChecksumVerification messages**: Checksums are now included in the request/response messages instead of being sent as separate messages after transfer

## Compatibility Notes


