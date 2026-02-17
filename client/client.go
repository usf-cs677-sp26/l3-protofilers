package main

import (
	"crypto/md5"
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func put(msgHandler *messages.MessageHandler, fileName string) int {
	fmt.Println("PUT", fileName)

	// Get file size and make sure it exists
	info, err := os.Stat(fileName)
	if err != nil {
		log.Fatalln(err)
	}

	// Compute checksum first
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	hash := md5.New()
	io.Copy(hash, file)
	file.Close()
	checksum := hash.Sum(nil)

	// Tell the server we want to store this file (send only the base name)
	baseName := filepath.Base(fileName)
	msgHandler.SendStorageRequest(baseName, uint64(info.Size()), checksum)
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	// Stream file data to server
	file, _ = os.Open(fileName)
	io.CopyN(msgHandler, file, info.Size())
	file.Close()
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	fmt.Println("Storage complete!")
	return 0
}

func get(msgHandler *messages.MessageHandler, fileName string, dir string) int {
	fmt.Println("GET", fileName)

	msgHandler.SendRetrievalRequest(fileName)
	ok, _, size, serverCheck := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		return 1
	}

	localPath := filepath.Join(dir, fileName)
	file, err := os.OpenFile(localPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
		return 1
	}

	hash := md5.New()
	w := io.MultiWriter(file, hash)
	io.CopyN(w, msgHandler, int64(size))
	file.Close()

	clientCheck := hash.Sum(nil)
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

	start := time.Now()
	var exitCode int
	if action == "put" {
		exitCode = put(msgHandler, fileName)
	} else if action == "get" {
		exitCode = get(msgHandler, fileName, dir)
	}
	fmt.Printf("Transfer took %v\n", time.Since(start))
	os.Exit(exitCode)
}
