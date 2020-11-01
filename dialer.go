package gortsplib

import (
	"net"
	"time"

	"github.com/aler9/gortsplib/base"
)

// DefaultDialer is the default dialer, used by DialRead and DialPublish.
var DefaultDialer = Dialer{}

// DialRead connects to the address and starts reading all tracks.
func DialRead(address string, proto StreamProtocol) (*ConnClient, error) {
	return DefaultDialer.DialRead(address, proto)
}

// DialPublish connects to the address and starts publishing the tracks.
func DialPublish(address string, proto StreamProtocol, tracks Tracks) (*ConnClient, error) {
	return DefaultDialer.DialPublish(address, proto, tracks)
}

// Dialer allows to connect to a server and read or publish tracks.
type Dialer struct {
	// (optional) timeout of read operations.
	// It defaults to 10 seconds
	ReadTimeout time.Duration

	// (optional) timeout of write operations.
	// It defaults to 5 seconds
	WriteTimeout time.Duration

	// (optional) read buffer count.
	// If greater than 1, allows to pass buffers to routines different than the one
	// that is reading frames.
	// It defaults to 1
	ReadBufferCount int

	// (optional) function used to initialize the TCP client.
	// It defaults to net.DialTimeout
	DialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)

	// (optional) function used to initialize UDP listeners.
	// It defaults to net.ListenPacket
	ListenPacket func(network, address string) (net.PacketConn, error)
}

// DialRead connects to the address and starts reading all tracks.
func (d Dialer) DialRead(address string, proto StreamProtocol) (*ConnClient, error) {
	u, err := base.ParseURL(address)
	if err != nil {
		return nil, err
	}

	conn, err := NewConnClient(ConnClientConf{
		Host:            u.Host(),
		ReadTimeout:     d.ReadTimeout,
		WriteTimeout:    d.WriteTimeout,
		ReadBufferCount: d.ReadBufferCount,
		DialTimeout:     d.DialTimeout,
		ListenPacket:    d.ListenPacket,
	})
	if err != nil {
		return nil, err
	}

	_, err = conn.Options(u)
	if err != nil {
		conn.Close()
		return nil, err
	}

	tracks, res, err := conn.Describe(u)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if res.StatusCode >= base.StatusMovedPermanently &&
		res.StatusCode <= base.StatusUseProxy {
		conn.Close()
		return d.DialRead(res.Header["Location"][0], proto)
	}

	if proto == StreamProtocolUDP {
		for _, track := range tracks {
			_, err := conn.SetupUDP(u, TransportModePlay, track, 0, 0)
			if err != nil {
				return nil, err
			}
		}

	} else {
		for _, track := range tracks {
			_, err := conn.SetupTCP(u, TransportModePlay, track)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}
	}

	_, err = conn.Play(u)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// DialPublish connects to the address and starts publishing the tracks.
func (d Dialer) DialPublish(address string, proto StreamProtocol, tracks Tracks) (*ConnClient, error) {
	u, err := base.ParseURL(address)
	if err != nil {
		return nil, err
	}

	conn, err := NewConnClient(ConnClientConf{
		Host:            u.Host(),
		ReadTimeout:     d.ReadTimeout,
		WriteTimeout:    d.WriteTimeout,
		ReadBufferCount: d.ReadBufferCount,
		DialTimeout:     d.DialTimeout,
		ListenPacket:    d.ListenPacket,
	})
	if err != nil {
		return nil, err
	}

	_, err = conn.Options(u)
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = conn.Announce(u, tracks)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if proto == StreamProtocolUDP {
		for _, track := range tracks {
			_, err = conn.SetupUDP(u, TransportModeRecord, track, 0, 0)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}

	} else {
		for _, track := range tracks {
			_, err = conn.SetupTCP(u, TransportModeRecord, track)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}
	}

	_, err = conn.Record(u)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}
