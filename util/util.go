package util

import (
	"crypto/md5"
	"io"
	"log"
	"reflect"
	"syscall"
)

func VerifyChecksum(serverCheck []byte, clientCheck []byte) bool {
	log.Printf("Server checksum: %x\n", serverCheck)
	log.Printf("Client checksum: %x\n", clientCheck)
	if reflect.DeepEqual(clientCheck, serverCheck) {
		log.Println("Checksums match")
		return true
	} else {
		log.Println("Checksums DO NOT match")
		return false
	}
}

// CopyN copies exactly n bytes from src to dst using the provided buffer,
// avoiding the 32KB default buffer in io.CopyN.
func CopyN(dst io.Writer, src io.Reader, n int64, buf []byte) (int64, error) {
	return io.CopyBuffer(dst, io.LimitReader(src, n), buf)
}

// HasDiskSpace reports whether path has at least needed bytes of free space available.
func HasDiskSpace(path string, needed uint64) bool {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return true // can't determine; let the write fail naturally
	}
	return stat.Bavail*uint64(stat.Bsize) >= needed
}

// CopyWithChecksum copies exactly n bytes from src to dst while computing
// an MD5 checksum over the data. Returns the checksum of the copied bytes.
func CopyWithChecksum(dst io.Writer, src io.Reader, n int64) ([]byte, error) {
	h := md5.New()
	w := io.MultiWriter(dst, h)
	buf := make([]byte, 1048576) // 1 MB copy buffer
	_, err := CopyN(w, src, n, buf)
	return h.Sum(nil), err
}
