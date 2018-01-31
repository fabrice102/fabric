package streamio

import "testing"

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
		copy(m.messages, m.messages[1:])
		return retBuf, nil
	}

	return nil, nil

}

func TestRead(t *testing.T) {
	var buf []byte
	//var nRead int
	//var err error

	mrr := newMockMessageReader([][]byte{
		{}
	})
	r.buf = []byte("test")

	//create a fake message reader

	_, _ = r.Read(buf)
	//nRead, err = r.Read(buf)

	//show nRead

}
