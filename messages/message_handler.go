package messages

import (
	"bufio"
	"encoding/binary"
	"log"
	"net"

	"google.golang.org/protobuf/proto"
)

const bufSize = 1048576 // 1 MB

type MessageHandler struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func NewMessageHandler(conn net.Conn) *MessageHandler {
	m := &MessageHandler{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, bufSize),
		writer: bufio.NewWriterSize(conn, bufSize),
	}

	return m
}

func (m *MessageHandler) ReadN(buf []byte) error {
	bytesRead := uint64(0)
	for bytesRead < uint64(len(buf)) {
		n, err := m.reader.Read(buf[bytesRead:])
		if err != nil {
			return err
		}
		bytesRead += uint64(n)
	}
	return nil
}

func (m *MessageHandler) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *MessageHandler) Write(p []byte) (n int, err error) {
	return m.writer.Write(p)
}

func (m *MessageHandler) WriteN(buf []byte) error {
	bytesWritten := uint64(0)
	for bytesWritten < uint64(len(buf)) {
		n, err := m.writer.Write(buf[bytesWritten:])
		if err != nil {
			return err
		}
		bytesWritten += uint64(n)
	}
	return nil
}

func (m *MessageHandler) Flush() error {
	return m.writer.Flush()
}

func (m *MessageHandler) Send(wrapper *Wrapper) error {
	serialized, err := proto.Marshal(wrapper)
	if err != nil {
		return err
	}

	prefix := make([]byte, 8)
	binary.LittleEndian.PutUint64(prefix, uint64(len(serialized)))
	m.WriteN(prefix)
	m.WriteN(serialized)

	return m.Flush()
}

func (m *MessageHandler) Receive() (*Wrapper, error) {
	prefix := make([]byte, 8)
	m.ReadN(prefix)

	payloadSize := binary.LittleEndian.Uint64(prefix)
	payload := make([]byte, payloadSize)
	m.ReadN(payload)

	wrapper := &Wrapper{}
	err := proto.Unmarshal(payload, wrapper)
	return wrapper, err
}

func (m *MessageHandler) Close() {
	m.writer.Flush()
	m.conn.Close()
}

func (m *MessageHandler) SendStorageRequest(fileName string, size uint64) error {
	msg := StorageRequest{FileName: fileName, Size: size}
	wrapper := &Wrapper{
		Msg: &Wrapper_StorageReq{StorageReq: &msg},
	}
	return m.Send(wrapper)
}

func (m *MessageHandler) SendRetrievalRequest(fileName string) error {
	msg := RetrievalRequest{FileName: fileName}
	wrapper := &Wrapper{
		Msg: &Wrapper_RetrievalReq{RetrievalReq: &msg},
	}
	return m.Send(wrapper)
}

func (m *MessageHandler) SendChecksumVerification(checksum []byte) error {
	checkMsg := ChecksumVerification{Checksum: checksum}
	checkWrapper := &Wrapper{
		Msg: &Wrapper_Checksum{Checksum: &checkMsg},
	}
	return m.Send(checkWrapper)
}

func (m *MessageHandler) SendResponse(ok bool, str string) error {
	msg := Response{Ok: ok, Message: str}
	wrapper := &Wrapper{
		Msg: &Wrapper_Response{Response: &msg},
	}

	return m.Send(wrapper)
}

func (m *MessageHandler) SendRetrievalResponse(ok bool, str string, size uint64) error {
	resp := Response{Ok: ok, Message: str}
	msg := RetrievalResponse{Resp: &resp, Size: size}
	wrapper := &Wrapper{
		Msg: &Wrapper_RetrievalResp{RetrievalResp: &msg},
	}

	return m.Send(wrapper)
}

func (m *MessageHandler) ReceiveResponse() (bool, string) {
	resp, err := m.Receive()
	if err != nil {
		return false, ""
	}

	log.Println(resp.GetResponse().Message)
	return resp.GetResponse().Ok, resp.GetResponse().Message
}

func (m *MessageHandler) ReceiveChecksum() []byte {
	msg, _ := m.Receive()
	return msg.GetChecksum().Checksum
}

func (m *MessageHandler) ReceiveRetrievalResponse() (bool, string, uint64) {
	resp, err := m.Receive()
	if err != nil {
		return false, "", 0
	}

	rr := resp.GetRetrievalResp().GetResp()
	log.Println(rr.Message)
	return rr.Ok, rr.Message, resp.GetRetrievalResp().Size
}
