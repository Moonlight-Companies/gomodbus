package transport

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Moonlight-Companies/gomodbus/common"
	"github.com/Moonlight-Companies/gomodbus/logging"
)

// TCPTransport implements the common.Transport interface for Modbus TCP
type TCPTransport struct {
	logger          common.LoggerInterface
	host            string
	port            int
	timeout         time.Duration
	conn            net.Conn
	reader          io.Reader
	writer          io.Writer
	mutex           sync.Mutex
	connected       bool
	closeOnce       sync.Once
	transactionPool *TransactionPool
	writeChan       chan *Transaction
	done            chan struct{}
}

// TCPTransportOption is a function that configures a TCPTransport
type TCPTransportOption func(*TCPTransport)

// WithPort sets the TCP port
func WithPort(port int) TCPTransportOption {
	return func(t *TCPTransport) {
		t.port = port
	}
}

// WithTimeout sets the timeout duration
func WithTimeoutOption(timeout time.Duration) TCPTransportOption {
	return func(t *TCPTransport) {
		t.timeout = timeout
	}
}

// WithReader sets the reader
func WithReader(reader io.Reader) TCPTransportOption {
	return func(t *TCPTransport) {
		t.reader = reader
	}
}

// WithWriter sets the writer
func WithWriter(writer io.Writer) TCPTransportOption {
	return func(t *TCPTransport) {
		t.writer = writer
	}
}

// WithTransportLogger sets the logger for the transport
func WithTransportLogger(logger common.LoggerInterface) TCPTransportOption {
	return func(t *TCPTransport) {
		t.logger = logger
	}
}

// NewTCPTransport creates a new TCPTransport
func NewTCPTransport(host string, options ...TCPTransportOption) *TCPTransport {
	t := &TCPTransport{
		logger:          logging.NewLogger(),
		host:            host,
		port:            common.DefaultTCPPort,
		timeout:         30 * time.Second,
		connected:       false,
		transactionPool: NewTransactionPool(),
		writeChan:       make(chan *Transaction, 100),
		done:            make(chan struct{}),
	}

	for _, option := range options {
		option(t)
	}

	return t
}

// WithLogger sets the logger for the transport and returns the modified transport
func (t *TCPTransport) WithLogger(logger common.LoggerInterface) common.Transport {
	t.logger = logger
	return t
}

// Connect establishes a connection to the Modbus TCP server
func (t *TCPTransport) Connect(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.connected {
		return common.ErrAlreadyConnected
	}

	t.logger.Info(ctx, "Connecting to Modbus TCP server at %s:%d", t.host, t.port)

	// Get deadline from context or use default timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(t.timeout)
	}

	// Connect with timeout
	dialer := net.Dialer{
		Timeout: time.Until(deadline),
	}

	addr := fmt.Sprintf("%s:%d", t.host, t.port)
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.logger.Error(ctx, "Failed to connect to %s: %v", addr, err)
		return err
	}

	t.conn = conn

	// If no custom reader/writer was provided, use the connection
	if t.reader == nil {
		t.reader = t.conn
	}
	if t.writer == nil {
		t.writer = t.conn
	}

	t.connected = true

	t.logger.Info(ctx, "Connected to Modbus TCP server at %s:%d", t.host, t.port)

	// Start the read and write goroutines
	go t.readLoop()
	go t.writeLoop()

	return nil
}

// Disconnect closes the connection to the Modbus TCP server
func (t *TCPTransport) Disconnect(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.connected {
		return nil
	}

	t.logger.Info(ctx, "Disconnecting from Modbus TCP server")

	// Signal goroutines to exit
	close(t.done)

	var err error
	t.closeOnce.Do(func() {
		// Close the transaction pool first so pending transactions get cancelled
		t.transactionPool.Close()

		// Close the connection
		if t.conn != nil {
			err = t.conn.Close()
		}
	})

	t.connected = false

	t.logger.Info(ctx, "Disconnected from Modbus TCP server")
	return err
}

// IsConnected returns true if connected to the server
func (t *TCPTransport) IsConnected() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.connected
}

// readLoop continuously reads from the connection and handles responses
func (t *TCPTransport) readLoop() {
	ctx := context.Background()
	t.logger.Debug(ctx, "Starting read loop")

	defer func() {
		t.logger.Debug(ctx, "Exiting read loop")
		t.setDisconnected(fmt.Errorf("read loop exited"))
	}()

	for {
		select {
		case <-t.done:
			return
		default:
			// Read the response header (7 bytes)
			header := make([]byte, common.TCPHeaderLength)
			_, err := io.ReadFull(t.reader, header)
			if err != nil {
				t.logger.Error(ctx, "Error reading header: %v", err)
				t.setDisconnected(fmt.Errorf("read error: %w", err))
				return
			}

			// If logger implements Hexdump and we're at trace level, log the header
			if hexLogger, ok := t.logger.(common.LoggerInterfaceHexdump); ok {
				hexLogger.Hexdump(ctx, header)
			}

			// Parse the header
			transactionID := common.TransactionID(binary.BigEndian.Uint16(header[0:2]))
			protocolID := common.ProtocolID(binary.BigEndian.Uint16(header[2:4]))
			length := binary.BigEndian.Uint16(header[4:6])
			unitID := common.UnitID(header[6])

			t.logger.Debug(ctx, "Received response: txID=%d, length=%d", transactionID, length)

			// Check ProtocolID
			if protocolID != common.TCPProtocolIdentifier {
				t.logger.Error(ctx, "Invalid protocol ID: %d", protocolID)
				t.processError(transactionID, common.ErrInvalidProtocolHeader)
				continue
			}

			// Length is the number of bytes following (unit ID + PDU)
			// We already read the unit ID, so we need length-1 more bytes
			bodyLength := int(length) - 1
			if bodyLength <= 0 {
				t.logger.Error(ctx, "Invalid response length: %d", length)
				t.processError(transactionID, common.ErrInvalidResponseLength)
				continue
			}

			// Read the function code and data
			body := make([]byte, bodyLength)
			_, err = io.ReadFull(t.reader, body)
			if err != nil {
				t.logger.Error(ctx, "Error reading body: %v", err)
				t.processError(transactionID, fmt.Errorf("read body error: %w", err))
				t.setDisconnected(err)
				return
			}

			// If logger implements Hexdump and we're at trace level, log the body
			if hexLogger, ok := t.logger.(common.LoggerInterfaceHexdump); ok {
				hexLogger.Hexdump(ctx, body)
			}

			// Create a response
			functionCode := common.FunctionCode(body[0])
			responseData := body[1:]
			response := NewResponse(transactionID, unitID, functionCode, responseData)

			// Find and complete the transaction
			tx, ok := t.transactionPool.Release(transactionID)
			if !ok {
				t.logger.Warn(ctx, "Received response for unknown transaction ID: %d", transactionID)
				continue
			}

			t.logger.Debug(ctx, "Completing transaction %d", transactionID)
			// Complete the transaction with the response
			tx.Complete(response, nil)
		}
	}
}

// writeLoop continuously processes requests from the writeChan
func (t *TCPTransport) writeLoop() {
	ctx := context.Background()
	t.logger.Debug(ctx, "Starting write loop")

	defer func() {
		t.logger.Debug(ctx, "Exiting write loop")
		t.setDisconnected(fmt.Errorf("write loop exited"))
	}()

	for {
		select {
		case <-t.done:
			return
		case tx := <-t.writeChan:
			// Check if the transaction is still valid
			select {
			case <-tx.Context().Done():
				t.logger.Debug(ctx, "Transaction %d was cancelled before writing",
					tx.Request.GetTransactionID())
				continue
			default:
				// Transaction is still valid
			}

			t.logger.Debug(ctx, "Writing request for transaction %d",
				tx.Request.GetTransactionID())

			// Encode the request
			data, err := tx.Request.Encode()
			if err != nil {
				t.logger.Error(ctx, "Error encoding request: %v", err)
				tx.Complete(nil, err)
				continue
			}

			// If logger implements Hexdump and we're at trace level, log the encoded request
			if hexLogger, ok := t.logger.(common.LoggerInterfaceHexdump); ok {
				hexLogger.Hexdump(ctx, data)
			}

			// Write the request
			_, err = t.writer.Write(data)
			if err != nil {
				t.logger.Error(ctx, "Error writing request: %v", err)
				tx.Complete(nil, err)
				t.setDisconnected(fmt.Errorf("write error: %w", err))
				return
			}

			t.logger.Debug(ctx, "Wrote request for transaction %d",
				tx.Request.GetTransactionID())
		}
	}
}

// processError handles errors for a specific transaction
func (t *TCPTransport) processError(txID common.TransactionID, err error) {
	ctx := context.Background()
	// Try to find the transaction and complete it with error
	if tx, ok := t.transactionPool.Release(txID); ok {
		t.logger.Debug(ctx, "Processing error for transaction %d: %v", txID, err)
		tx.Complete(nil, err)
	} else {
		t.logger.Warn(ctx, "Error for unknown transaction %d: %v", txID, err)
	}
}

// setDisconnected marks the transport as disconnected
func (t *TCPTransport) setDisconnected(err error) {
	ctx := context.Background()
	t.mutex.Lock()
	wasConnected := t.connected
	t.connected = false
	t.mutex.Unlock()

	if wasConnected {
		t.logger.Error(ctx, "Transport disconnected: %v", err)
	}
}

// Send sends a request and returns the response
func (t *TCPTransport) Send(ctx context.Context, request common.Request) (common.Response, error) {
	if !t.IsConnected() {
		return nil, common.ErrNotConnected
	}

	// txid pre allocation should be 0
	t.logger.Debug(ctx, "Sending request: function=%d", request.GetPDU().FunctionCode)

	// Create a transaction and add it to the pool
	tx, err := t.transactionPool.Place(ctx, request)
	if err != nil {
		t.logger.Error(ctx, "Failed to create transaction: %v", err)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	t.logger.Debug(ctx, "Created transaction %d", request.GetTransactionID())

	// Send the transaction to the write loop
	select {
	case t.writeChan <- tx:
		t.logger.Debug(ctx, "Queued transaction %d for writing", request.GetTransactionID())
	case <-ctx.Done():
		// Context cancelled before we could queue
		t.logger.Debug(ctx, "Context cancelled before queueing transaction %d",
			request.GetTransactionID())
		t.transactionPool.Release(request.GetTransactionID())
		return nil, ctx.Err()
	case <-t.done:
		// Transport is shutting down
		t.logger.Debug(ctx, "Transport shutting down, cancelling transaction %d",
			request.GetTransactionID())
		t.transactionPool.Release(request.GetTransactionID())
		return nil, common.ErrTransportClosing
	}

	// Wait for the response
	select {
	case response := <-tx.ResponseCh:
		t.logger.Debug(ctx, "Received response for transaction %d", request.GetTransactionID())
		return response, nil
	case err := <-tx.ErrCh:
		t.logger.Debug(ctx, "Received error for transaction %d: %v",
			request.GetTransactionID(), err)
		return nil, err
	case <-ctx.Done():
		// Context cancelled while waiting for response
		t.logger.Debug(ctx, "Context cancelled while waiting for transaction %d",
			request.GetTransactionID())
		// Transaction will be cleaned up by timeout monitor
		return nil, ctx.Err()
	}
}
