package main

import (
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func put(msgHandler *messages.MessageHandler, fileName string) int {
	fmt.Println("PUT", fileName)

	// Get file size and make sure it exists
	info, err := os.Stat(fileName)
	if err != nil {
		log.Fatalln(err)
	}

	// Pre-compute checksum before sending the request
	file, _ := os.Open(fileName)
	checksum, _ := util.CopyWithChecksum(io.Discard, file, info.Size())
	file.Close()
	fmt.Printf("File: %s, Size: %d bytes, Checksum: %x\n", fileName, info.Size(), checksum)

	// Tell the server we want to store this file (include checksum upfront)
	msgHandler.SendStorageRequest(filepath.Base(fileName), uint64(info.Size()), checksum)
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	file, _ = os.Open(fileName)
	util.CopyWithChecksum(msgHandler, file, info.Size())
	file.Close()
	msgHandler.Flush() // flush buffered file data before waiting for server response

	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	fmt.Println("Storage complete!")
	return 0
}

func get(msgHandler *messages.MessageHandler, fileName string, dir string) int {
	msgHandler.SendRetrievalRequest(fileName)
	ok, _, size, serverCheck := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		return 1
	}
	fmt.Printf("GET %s (%d bytes)\n", fileName, size)

	if !util.HasDiskSpace(dir, size) {
		log.Printf("Insufficient disk space: need %d bytes\n", size)
		return 1
	}

	outPath := filepath.Join(dir, fileName)
	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
		return 1
	}

	clientCheck, _ := util.CopyWithChecksum(file, msgHandler, int64(size))
	file.Close()

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully retrieved file.")
	} else {
		log.Println("FAILED to retrieve file. Invalid checksum.")
	}

	return 0
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Not enough arguments. Usage: %s server:port put|get file-name [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	host := os.Args[1]
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	msgHandler := messages.NewMessageHandler(conn)
	defer conn.Close()

	action := strings.ToLower(os.Args[2])
	if action != "put" && action != "get" {
		log.Fatalln("Invalid action", action)
	}

	fileName := os.Args[3]

	dir := "."
	if len(os.Args) >= 5 {
		dir = os.Args[4]
	}
	openDir, err := os.Open(dir)
	if err != nil {
		log.Fatalln(err)
	}
	openDir.Close()

	if action == "put" {
		os.Exit(put(msgHandler, fileName))
	} else if action == "get" {
		os.Exit(get(msgHandler, fileName, dir))
	}
}
