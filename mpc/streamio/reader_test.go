package streamio

import (
	"fmt"
	"testing"
)

type MockMessageReader struct {
	messages [][]byte
}

//read each message and move it from the stack
func newMockMessageReader(messages [][]byte) *MockMessageReader {
	return &MockMessageReader{
		messages: messages,
	}
}

func (m *MockMessageReader) Read() ([]byte, error) {
	var retBuf []byte

	if len(m.messages) > 0 {
		//remove message from buffer
		retBuf = m.messages[0]
		m.messages = m.messages[1:]
		return retBuf, nil
	}

	return nil, nil

}

func TestRead(t *testing.T) {
	var buf []byte
	buf = make([]byte, 12)
	//var nRead int
	//var err error

	mrr := newMockMessageReader([][]byte{
		[]byte("hello"),
		[]byte("world"),
	})

	reader := NewReader(mrr)
	_, _ = reader.Read(buf)
	//buf = []byte("test")

	//create a fake message reader

	//buf, _ = mrr.Read()
	//_ = buf
	//nRead, err = r.Read(buf)
	fmt.Printf("Read buffer: [%v]\n", buf)
	//show nRead

}
