package client

import (
	"io"

	"github.com/Moonlight-Companies/gomodbus/common"
	"github.com/Moonlight-Companies/gomodbus/transport"
)

// TCPClient is a Modbus TCP client
type TCPClient struct {
	*BaseClient
	tcpTransport *transport.TCPTransport
}

// TCPOption is a function that configures a TCPClient
type TCPOption func(*TCPClient)

// WithTCPLogger sets the logger for the TCP client
func WithTCPLogger(logger common.LoggerInterface) TCPOption {
	return func(c *TCPClient) {
		c.BaseClient = c.BaseClient.WithLogger(logger).(*BaseClient)
	}
}

// WithTCPUnitID sets the unit ID for the TCP client
func WithTCPUnitID(unitID common.UnitID) TCPOption {
	return func(c *TCPClient) {
		c.BaseClient = NewBaseClient(
			c.tcpTransport,
			WithUnitID(unitID),
			WithLogger(c.BaseClient.logger),
			WithProtocol(c.BaseClient.protocol),
		)
	}
}

// NewTCPClient creates a new Modbus TCP client
func NewTCPClient(host string, options ...transport.TCPTransportOption) *TCPClient {
	// Create the TCP transport
	tcpTransport := transport.NewTCPTransport(host, options...)
	
	// Create the base client with the transport
	baseClient := NewBaseClient(tcpTransport)
	
	// Create and return the TCP client
	return &TCPClient{
		BaseClient:   baseClient,
		tcpTransport: tcpTransport,
	}
}

// WithOptions applies the given options to the TCPClient
func (c *TCPClient) WithOptions(options ...TCPOption) *TCPClient {
	// Apply the options
	for _, option := range options {
		option(c)
	}
	return c
}

// WithUnitID sets the unit ID for the client and returns the client
// (Deprecated in favor of WithOptions(WithTCPUnitID(unitID)))
func (c *TCPClient) WithUnitID(unitID common.UnitID) *TCPClient {
	return c.WithOptions(WithTCPUnitID(unitID))
}

// WithLogger sets the logger for the client and returns the client
// (Deprecated in favor of WithOptions(WithTCPLogger(logger)))
func (c *TCPClient) WithLogger(logger common.LoggerInterface) common.Client {
	return c.WithOptions(WithTCPLogger(logger))
}

// FromReaderWriter creates a new client that reads from the given reader and writes to the given writer
// This is useful for testing or for using custom transports
func FromReaderWriter(reader io.Reader, writer io.Writer) *TCPClient {
	tcpTransport := transport.NewTCPTransport("test",
		transport.WithReader(reader),
		transport.WithWriter(writer),
	)
	baseClient := NewBaseClient(tcpTransport)

	return &TCPClient{
		BaseClient:   baseClient,
		tcpTransport: tcpTransport,
	}
}