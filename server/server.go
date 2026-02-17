package main

import (
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	log.Println("Attempting to store", request.FileName)
	file, err := os.OpenFile(request.FileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	if !util.HasDiskSpace(".", request.Size) {
		file.Close()
		os.Remove(request.FileName)
		msgHandler.SendResponse(false, fmt.Sprintf("Insufficient disk space: need %d bytes", request.Size))
		return
	}

	msgHandler.SendResponse(true, "OK-Server is ready for data")
	start := time.Now()
	serverCheck, _ := util.CopyWithChecksum(file, msgHandler, int64(request.Size))
	elapsed := time.Since(start)
	file.Close()

	clientCheck := msgHandler.ReceiveChecksum()

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully stored file.")
		msgHandler.SendResponse(true, fmt.Sprintf("Storage complete: checksum verified, time: %v", elapsed))
	} else {
		log.Println("FAILED to store file. Invalid checksum.")
		os.Remove(request.FileName)
		msgHandler.SendResponse(false, fmt.Sprintf("Storage failed: checksum mismatch, time: %v", elapsed))
	}
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	log.Println("Attempting to retrieve", request.FileName)

	// Get file size and make sure it exists
	info, err := os.Stat(request.FileName)
	if err != nil {
		log.Println("File not found:", err)
		msgHandler.SendRetrievalResponse(false, err.Error(), 0)
		return
	}

	msgHandler.SendRetrievalResponse(true, fmt.Sprintf("Server is ready to send file: %s, Size: %d bytes", request.FileName, info.Size()), uint64(info.Size()))

	file, _ := os.Open(request.FileName)
	start := time.Now()
	checksum, _ := util.CopyWithChecksum(msgHandler, file, info.Size())
	elapsed := time.Since(start)
	file.Close()
	msgHandler.Flush()

	msgHandler.SendChecksumVerification(checksum)
	msgHandler.SendRetrievalResponse(true, fmt.Sprintf("Retrieval complete: %s, time: %v", request.FileName, elapsed), uint64(info.Size()))

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
			msgHandler.SendResponse(true, "Server is terminating connection")
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
