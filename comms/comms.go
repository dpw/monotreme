package comms

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net"
	"sync"

	"github.com/dpw/monotreme/propagation"
	. "github.com/dpw/monotreme/rudiments"
)

const GUID_LEN uint = 8

func newNodeID() NodeID {
	bs := make([]byte, GUID_LEN)
	_, err := rand.Read(bs)
	if err != nil {
		panic("unable to get random bytes for NodeID")
	}
	return NodeID(hex.EncodeToString(bs))
}

type NodeDaemon struct {
	us       NodeID
	listener net.Listener

	lock         sync.Mutex
	connectivity *propagation.Connectivity
}

func NewNodeDaemon(bindAddr string) (*NodeDaemon, error) {
	us := newNodeID()

	l, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}

	nd := &NodeDaemon{
		us:           us,
		listener:     l,
		connectivity: propagation.NewConnectivity(us),
	}

	go nd.acceptConnections()
	return nd, nil
}

func (nd *NodeDaemon) acceptConnections() {
	for {
		conn, err := nd.listener.Accept()
		if err != nil {
			// XXX
			log.Println(err)
			return
		}

		go nd.handleConnection(conn)
	}
}

func (nd *NodeDaemon) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go nd.handleConnection(conn)
	return nil
}

type connection struct {
	nd        *NodeDaemon
	conn      net.Conn
	closeOnce sync.Once
	cancel    chan struct{}
	toSend    chan struct{}

	// protected by the NodeDaemon lock
	link *propagation.Link
}

func (nd *NodeDaemon) handleConnection(conn net.Conn) {
	c := connection{
		nd:     nd,
		conn:   conn,
		cancel: make(chan struct{}),
		toSend: make(chan struct{}, 1),
	}

	go func() {
		err := c.writeSide()
		if c.close() && err != nil && err != io.EOF {
			log.Println(err)
		}
	}()

	err := c.readSide()
	if c.close() && err != nil && err != io.EOF {
		log.Println(err)
	}
}

func (c *connection) writeSide() error {
	w := newWriter(c.conn)

	writeNodeID(w, c.nd.us)
	if err := w.endMessage(); err != nil {
		return err
	}

	for {
		select {
		case <-c.cancel:
			return nil
		case <-c.toSend:
		}

		if err := c.writePending(w); err != nil {
			return err
		}
	}
}

func (c *connection) writePending(w *writer) error {
	var propUpdates map[*propagation.Propagation][]propagation.Update

	func() {
		c.nd.lock.Lock()
		defer c.nd.lock.Unlock()
		propUpdates = c.link.Outgoing()
	}()

	for prop, updates := range propUpdates {
		writeConnectivityUpdates(w, updates)
		if err := w.endMessage(); err != nil {
			return err
		}

		func() {
			c.nd.lock.Lock()
			defer c.nd.lock.Unlock()
			c.link.Delivered(prop, updates)
		}()
	}

	return nil
}

func (c *connection) readSide() error {
	r := newReader(c.conn)

	them := readNodeID(r)
	if err := r.endMessage(); err != nil {
		return err
	}

	func() {
		c.nd.lock.Lock()
		defer c.nd.lock.Unlock()
		c.link = c.nd.connectivity.Link(them)
		c.link.SetPendingFunc(func() {
			select {
			case c.toSend <- struct{}{}:
			}
		})
	}()

	for {
		updates := readConnectivityUpdates(r)
		if err := r.endMessage(); err != nil {
			return err
		}

		func() {
			c.nd.lock.Lock()
			defer c.nd.lock.Unlock()
			c.link.Incoming(c.nd.connectivity.ConnectivityPropagation(), updates)
			log.Println(c.nd.connectivity.Dump())
		}()
	}
}

func (c *connection) close() bool {
	closed := false
	c.closeOnce.Do(func() {
		c.conn.Close()
		close(c.cancel)

		c.nd.lock.Lock()
		defer c.nd.lock.Unlock()
		c.link.Close()

		closed = true
	})

	return closed
}
