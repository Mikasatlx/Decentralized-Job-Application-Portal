package udp

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.dedis.ch/cs438/transport"
)

const bufSize = 65000

var counter uint32 = 1024

// NewUDP returns a new udp transport implementation.
func NewUDP() transport.Transport {
	return &UDP{}
}

// UDP implements a transport layer using UDP
//
// - implements transport.Transport
type UDP struct {
	conn *net.UDPConn
}

// CreateSocket implements transport.Transport
func (n *UDP) CreateSocket(address string) (transport.ClosableSocket, error) {
	//panic("to be implemented in HW0")
	if strings.HasSuffix(address, ":0") {
		address = address[:len(address)-2]
		port := atomic.AddUint32(&counter, 1)
		address = fmt.Sprintf("%s:%d", address, port)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	n.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &Socket{
		UDP:    *n,
		myAddr: address,
		ins:    Packets{},
		outs:   Packets{},
	}, nil
}

// Socket implements a network socket using UDP.
//
// - implements transport.Socket
// - implements transport.ClosableSocket
type Socket struct {
	UDP
	myAddr string
	ins    Packets
	outs   Packets
}

// Close implements transport.Socket. It returns an error if already closed.
func (s *Socket) Close() error {
	//panic("to be implemented in HW0")
	return s.conn.Close()
}

// Send implements transport.Socket
func (s *Socket) Send(dest string, pkt transport.Packet, timeout time.Duration) error {
	//panic("to be implemented in HW0")
	if timeout == 0 {
		timeout = math.MaxInt64
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	buf, err := pkt.Marshal()
	if err != nil {
		return err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		return err
	}
	_, err = s.conn.WriteToUDP(buf, udpAddr)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return transport.TimeoutError(timeout)
		}
		return err
	}

	s.outs.AddPacket(pkt)
	return nil
}

// Recv implements transport.Socket. It blocks until a packet is received, or
// the timeout is reached. In the case the timeout is reached, return a
// TimeoutErr.
func (s *Socket) Recv(timeout time.Duration) (transport.Packet, error) {
	//panic("to be implemented in HW0")
	pkt := transport.Packet{}
	buf := make([]byte, bufSize)

	if err := s.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return pkt, err
	}

	dataLen, _, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return pkt, transport.TimeoutError(timeout)
		}
		return pkt, err
	}

	if err := pkt.Unmarshal(buf[:dataLen]); err != nil {
		return pkt, err
	}

	s.ins.AddPacket(pkt)
	return pkt, nil
}

// GetAddress implements transport.Socket. It returns the address assigned. Can
// be useful in the case one provided a :0 address, which makes the system use a
// random free port.
func (s *Socket) GetAddress() string {
	//panic("to be implemented in HW0")
	return s.myAddr
}

// GetIns implements transport.Socket
func (s *Socket) GetIns() []transport.Packet {
	//panic("to be implemented in HW0")
	return s.ins.GetAllPackets()
}

// GetOuts implements transport.Socket
func (s *Socket) GetOuts() []transport.Packet {
	//panic("to be implemented in HW0")
	return s.outs.GetAllPackets()
}

type Packets struct {
	sync.RWMutex
	pkts []transport.Packet
}

// AddPacket add packets to array in thread safe manner
func (p *Packets) AddPacket(pkt transport.Packet) {
	p.Lock()
	defer p.Unlock()
	p.pkts = append(p.pkts, pkt)
}

// GetAllPackets deep-copies the packet array and return
func (p *Packets) GetAllPackets() []transport.Packet {
	p.RLock()
	defer p.RUnlock()
	copyPkts := make([]transport.Packet, len(p.pkts))
	for i, pkt := range p.pkts {
		copyPkts[i] = pkt.Copy()
	}
	return copyPkts
}
