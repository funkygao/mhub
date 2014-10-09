package broker

import (
	"fmt"
	"io"
)

// An intPayload implements proto.Payload, and is an int64 that
// formats itself and then prints itself into the payload.
type intPayload string

func newIntPayload(i int64) intPayload {
	return intPayload(fmt.Sprint(i))
}
func (ip intPayload) ReadPayload(r io.Reader) error {
	// not implemented
	return nil
}
func (ip intPayload) WritePayload(w io.Writer) error {
	_, err := w.Write([]byte(string(ip)))
	return err
}
func (i intPayload) Size() int {
	return len(i)
}
