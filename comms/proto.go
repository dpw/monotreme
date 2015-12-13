package comms

import (
	"bufio"
	"encoding/binary"
	"io"
	"reflect"

	"github.com/dpw/monotreme/propagation"
	. "github.com/dpw/monotreme/rudiments"
)

var end = binary.LittleEndian

type writer struct {
	*bufio.Writer
	err error
}

func newWriter(w io.Writer) *writer {
	return &writer{bufio.NewWriter(w), nil}
}

func (w *writer) write(val interface{}) {
	if w.err == nil {
		w.err = binary.Write(w, end, val)
	}
}

func (w *writer) writeArray(a interface{}, elemWriter func(*writer, interface{})) {
	av := reflect.ValueOf(a)
	len := av.Len()

	// XXX check that len does not exceed uint32 range
	w.write(uint32(len))

	for i := 0; i < len; i++ {
		elemWriter(w, av.Index(i).Interface())
	}
}

func (w *writer) Flush() error {
	if w.err == nil {
		w.err = w.Writer.Flush()
	}

	return w.err
}

type reader struct {
	*bufio.Reader
	err error
}

func newReader(r io.Reader) *reader {
	return &reader{bufio.NewReader(r), nil}
}

func (r *reader) read(val interface{}) {
	if r.err == nil {
		r.err = binary.Read(r, end, val)
	}
}

func (r *reader) readArray(el interface{}, elemReader func(*reader) interface{}) interface{} {
	var len uint32
	r.read(&len)

	s := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(el)),
		int(len), int(len))

	for i := 0; uint32(i) < len; i++ {
		s.Index(i).Set(reflect.ValueOf(elemReader(r)))
	}

	return s.Interface()
}

func writeNodeID(w *writer, n NodeID) {
	bytes := ([]byte)(n)
	w.write(uint16(len(bytes)))
	w.write(bytes)
}

func writeConnectivityUpdates(w *writer, updates []propagation.Update) {
	w.writeArray(updates, func(w *writer, el interface{}) {
		u := el.(propagation.Update)
		writeNodeID(w, u.Node)
		w.write(u.Version)
		w.writeArray(u.State, func(w *writer, nodeID interface{}) {
			writeNodeID(w, nodeID.(NodeID))
		})
	})
}

func readNodeID(r *reader) NodeID {
	var len uint16
	r.read(&len)
	bytes := make([]byte, len)
	r.read(bytes)
	return NodeID(bytes)
}

func readConnectivityUpdates(r *reader) []propagation.Update {
	return r.readArray(propagation.Update{}, func(r *reader) interface{} {
		update := propagation.Update{Node: readNodeID(r)}
		r.read(&update.Version)
		update.State = r.readArray(NodeID(""), func(r *reader) interface{} {
			return readNodeID(r)
		})
		return update
	}).([]propagation.Update)
}
