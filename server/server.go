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
	"syscall"
)

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	log.Println("Attempting to store", request.FileName)
	file, err := os.OpenFile(request.FileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Error:", err)
		msgHandler.SendResponse(false, err.Error())
		return
	}

	// Check if there is enough disk space
	var stat syscall.Statfs_t
	log.Printf("Checking disk space for file of size %d bytes\n", request.Size)
	if err := syscall.Statfs(".", &stat); err == nil {
		available := uint64(stat.Bavail) * uint64(stat.Bsize)
		if request.Size > available {
			log.Printf("Not enough disk space: need %d, have %d\n", request.Size, available)
			msgHandler.SendResponse(false, "Not enough disk space")
			file.Close()
			os.Remove(request.FileName)
			return
		}
	}

	msgHandler.SendResponse(true, "Ready for data")
	hash := md5.New()
	w := io.MultiWriter(file, hash)
	io.CopyN(w, msgHandler, int64(request.Size)) /* Write and checksum as we go */
	file.Close()

	serverCheck := hash.Sum(nil)
	clientCheck := request.Checksum

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully stored file.")
		msgHandler.SendResponse(true, "File stored successfully")
	} else {
		log.Println("FAILED to store file. Invalid checksum.")
		msgHandler.SendResponse(false, "Checksum mismatch")
	}
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	log.Println("Attempting to retrieve", request.FileName)

	// Get file size and make sure it exists
	info, err := os.Stat(request.FileName)
	if err != nil {
		log.Println("File not found:", err)
		msgHandler.SendRetrievalResponse(false, err.Error(), 0, nil)
		return
	}

	// Compute checksum before sending response
	file, err := os.Open(request.FileName)
	if err != nil {
		log.Println("Error opening file:", err)
		msgHandler.SendRetrievalResponse(false, err.Error(), 0, nil)
		return
	}
	hash := md5.New()
	io.Copy(hash, file)
	file.Close()
	checksum := hash.Sum(nil)

	msgHandler.SendRetrievalResponse(true, "Ready to send", uint64(info.Size()), checksum)

	// Stream file data
	file, _ = os.Open(request.FileName)
	io.CopyN(msgHandler, file, info.Size())
	file.Close()
}

func handleClient(msgHandler *messages.MessageHandler) {
	defer msgHandler.Close()

	for {
		wrapper, err := msgHandler.Receive()
		if err != nil {
			log.Println(err)
		}

		switch msg := wrapper.Msg.(type) {
		case *messages.Wrapper_StorageReq:
			handleStorage(msgHandler, msg.StorageReq)
			continue
		case *messages.Wrapper_RetrievalReq:
			handleRetrieval(msgHandler, msg.RetrievalReq)
			continue
		case nil:
			log.Println("Received an empty message, terminating client")
			return
		default:
			log.Printf("Unexpected message type: %T", msg)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Not enough arguments. Usage: %s port [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	port := os.Args[1]
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
	defer listener.Close()

	dir := "."
	if len(os.Args) >= 3 {
		dir = os.Args[2]
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalln("Failed to create directory:", err)
	}
	if err := os.Chdir(dir); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Listening on port:", port)
	fmt.Println("Download directory:", dir)
	for {
		if conn, err := listener.Accept(); err == nil {
			log.Println("Accepted connection", conn.RemoteAddr())
			handler := messages.NewMessageHandler(conn)
			go handleClient(handler)
		}
	}
}
