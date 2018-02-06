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

// Read implements the read function in io.Reader
func (r *Reader) Read(p []byte) (int, error) {
	//get the message from the buffer, otherwise call the MessageReader
	//define my own byte buffer
	//use the call to the function
	//create a buffer in the reader

	var err error
	//only read if the buffer is empty
	if r.buf == nil || len(r.buf) == 0 {
		r.buf, err = r.mr.Read()
	}

	if r.buf == nil || err != nil {
		r.buf = nil
		return 0, err
	}

	copy(p, r.buf)
	n := len(r.buf)
	r.buf = nil //clear the buffer

	//check its size nb - if it is longer than n, then return n bytes and save the rest of the nb-n bytes
	//if an error occurred, return it

	//copy from r.buf to p, max length of p
	//return size of data read

	return n, err
}
