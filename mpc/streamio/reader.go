package streamio

// MessageReader wraps a message-based Read function
type MessageReader interface {
	// Returns the next message.
	// If no message available, return nil, nil
	Read() ([]byte, error)
}

// Reader implements a streaming io.Reader from a MessageReader
type Reader struct {
	mr  MessageReader
	buf []byte
}

// NewReader creates a streaming reader from a MessageReader
func NewReader(mr MessageReader) *Reader {
	r := Reader{
		mr: mr,
	}
	return &r
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Read implements the read function in io.Reader
func (r *Reader) Read(p []byte) (int, error) {
	var err error

	// If no leftover data in r.buf, read the next message
	if len(r.buf) == 0 {
		r.buf, err = r.mr.Read()

		if r.buf == nil || err != nil {
			r.buf = nil
			return 0, err
		}
	}

	pLen := len(p)
	rBufLen := len(r.buf)

	copy(p, r.buf)
	n := min(pLen, rBufLen)

	if pLen < rBufLen {
		r.buf = r.buf[pLen:]
	} else {
		r.buf = nil
	}

	return n, err
}
