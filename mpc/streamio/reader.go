package streamio

import "math"

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
	var n int

	n = len(r.buf)
	pLen := len(p)

	//start by copying the leftover data from r.buf
	if r.buf != nil {
		copy(p, r.buf)

		if pLen < n {
			r.buf = r.buf[(len(p)):(len(r.buf))]
		} else {
			r.buf = nil
		}

		//we're OK either way
		// if pLen <= n {
		// 	return n, err
		// }
	}
	//here we have the previous message copied

	//how much more message can we fit
	pLeft := pLen - n

	if pLeft == 0 {
		return n, err
	}

	//only read if the buffer is empty
	//if r.buf == nil || len(r.buf) == 0 {
	r.buf, err = r.mr.Read()
	//}

	if r.buf == nil || err != nil {
		r.buf = nil
		return 0, err
	}

	//p may be smaller than the buffer we just read

	lNewMessage := (int)(math.Min((float64)(n+len(r.buf)), (float64)(pLen)))
	copy(p[n:lNewMessage], r.buf)
	//p = p[0:lNewMessage]
	for i := lNewMessage; i < pLen; i++ {
		p[i] = 0
	}
	n = (int)(lNewMessage)
	//len(r.buf)

	if pLeft < len(r.buf) {
		r.buf = r.buf[pLeft:(len(r.buf))]
	} else {
		r.buf = nil
	}

	//check its size nb - if it is longer than n, then return n bytes and save the rest of the nb-n bytes
	//if an error occurred, return it

	//copy from r.buf to p, max length of p
	//return size of data read

	return n, err
}
