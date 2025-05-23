package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Moonlight-Companies/gomodbus/common"
	"github.com/Moonlight-Companies/gomodbus/logging"
	"github.com/Moonlight-Companies/gomodbus/transport"
)

// TCPServer implements a Modbus TCP server
// Implements the Modbus TCP protocol as defined in the Modbus specification
// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 4 (Modbus Protocol Description)
// Ref: Modbus_Messaging_Implementation_Guide_V1_0b.pdf, Section 3 (Modbus TCP/IP Protocol)
type TCPServer struct {
	// Server binding configuration
	address      string
	port         int
	listener     net.Listener

	// Function code handlers map
	handlers     map[common.FunctionCode]common.HandlerFunc

	// Data storage
	defaultStore common.DataStore

	// Server state
	running      bool
	clients      map[string]net.Conn
	clientsMutex sync.RWMutex
	mutex        sync.RWMutex
	logger       common.LoggerInterface
	stopChan     chan struct{}

	// Protocol handler for processing requests
	protocol     *serverProtocolHandler
}

// TCPServerOption is a function type for configuring a TCPServer
type TCPServerOption func(*TCPServer)

// WithServerPort sets the TCP port for the server
func WithServerPort(port int) TCPServerOption {
	return func(s *TCPServer) {
		s.port = port
	}
}

// WithServerLogger sets the logger for the TCP server
func WithServerLogger(logger common.LoggerInterface) TCPServerOption {
	return func(s *TCPServer) {
		s.logger = logger
	}
}

// WithServerDataStore sets the data store for the TCP server
func WithServerDataStore(store common.DataStore) TCPServerOption {
	return func(s *TCPServer) {
		s.defaultStore = store
	}
}

// NewTCPServer creates a new Modbus TCP server
func NewTCPServer(address string, options ...TCPServerOption) *TCPServer {
	server := &TCPServer{
		address:      address,
		port:         common.DefaultTCPPort,
		handlers:     make(map[common.FunctionCode]common.HandlerFunc),
		defaultStore: NewMemoryStore(),
		logger:       logging.NewLogger(),
		clients:      make(map[string]net.Conn),
		protocol:     newServerProtocolHandler(),
	}

	// Apply options
	for _, option := range options {
		option(server)
	}

	// Setup default handlers based on data store
	server.setupDefaultHandlers()

	return server
}

// WithLogger sets the logger for the server
func (s *TCPServer) WithLogger(logger common.LoggerInterface) common.Server {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.logger = logger
	return s
}

// WithDataStore sets the data store for the server
func (s *TCPServer) WithDataStore(dataStore common.DataStore) common.Server {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.defaultStore = dataStore
	s.setupDefaultHandlers()
	return s
}

// setupDefaultHandlers configures handlers for standard Modbus functions
// Sets up handlers for all supported Modbus function codes as defined in the specification
// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6 (Function Codes)
func (s *TCPServer) setupDefaultHandlers() {
	// Clear existing handlers
	s.handlers = make(map[common.FunctionCode]common.HandlerFunc)

	// Read Coils (0x01)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.1
	s.SetHandler(common.FuncReadCoils, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleReadCoils(ctx, req, s.defaultStore)
	})

	// Read Discrete Inputs (0x02)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.2
	s.SetHandler(common.FuncReadDiscreteInputs, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleReadDiscreteInputs(ctx, req, s.defaultStore)
	})

	// Read Holding Registers (0x03)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.3
	s.SetHandler(common.FuncReadHoldingRegisters, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleReadHoldingRegisters(ctx, req, s.defaultStore)
	})

	// Read Input Registers (0x04)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.4
	s.SetHandler(common.FuncReadInputRegisters, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleReadInputRegisters(ctx, req, s.defaultStore)
	})

	// Write Single Coil (0x05)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.5
	s.SetHandler(common.FuncWriteSingleCoil, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleWriteSingleCoil(ctx, req, s.defaultStore)
	})

	// Write Single Register (0x06)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.6
	s.SetHandler(common.FuncWriteSingleRegister, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleWriteSingleRegister(ctx, req, s.defaultStore)
	})

	// Write Multiple Coils (0x0F)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.11
	s.SetHandler(common.FuncWriteMultipleCoils, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleWriteMultipleCoils(ctx, req, s.defaultStore)
	})

	// Write Multiple Registers (0x10)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.12
	s.SetHandler(common.FuncWriteMultipleRegisters, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleWriteMultipleRegisters(ctx, req, s.defaultStore)
	})

	// Read/Write Multiple Registers (0x17)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.17
	s.SetHandler(common.FuncReadWriteMultipleRegisters, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleReadWriteMultipleRegisters(ctx, req, s.defaultStore)
	})

	// Read Device Identification (0x2B)
	// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6.21
	s.SetHandler(common.FuncReadDeviceIdentification, func(ctx context.Context, req common.Request) (common.Response, error) {
		return s.protocol.HandleReadDeviceIdentification(ctx, req, s.defaultStore)
	})
}

// SetHandler sets the handler for a specific Modbus function code
func (s *TCPServer) SetHandler(functionCode common.FunctionCode, handler common.HandlerFunc) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handlers[functionCode] = handler
}

// Start starts the server
func (s *TCPServer) Start(ctx context.Context) error {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return fmt.Errorf("server already running")
	}

	addr := fmt.Sprintf("%s:%d", s.address, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mutex.Unlock()
		return err
	}

	s.listener = listener
	s.running = true
	s.stopChan = make(chan struct{})
	s.mutex.Unlock()

	s.logger.Info(ctx, "Modbus TCP server started on %s", addr)

	// Start accepting connections
	go s.acceptLoop(ctx)

	return nil
}

// Stop stops the server
func (s *TCPServer) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil // Already stopped
	}

	// Signal accept loop to stop
	close(s.stopChan)

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all client connections
	s.clientsMutex.Lock()
	for _, conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[string]net.Conn)
	s.clientsMutex.Unlock()

	s.running = false
	s.logger.Info(ctx, "Modbus TCP server stopped")
	return nil
}

// IsRunning returns true if the server is running
func (s *TCPServer) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// acceptLoop accepts incoming connections
func (s *TCPServer) acceptLoop(ctx context.Context) {
	for {
		// Check if we should stop
		select {
		case <-s.stopChan:
			return
		default:
			// Continue accepting
		}

		// Set accept deadline to allow checking for stop signal
		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))

		conn, err := s.listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// Timeout, just retry
				continue
			}

			// Check if we're shutting down
			select {
			case <-s.stopChan:
				return
			default:
				s.logger.Error(ctx, "Error accepting connection: %v", err)
				continue
			}
		}

		s.logger.Info(ctx, "New client connected: %s", conn.RemoteAddr().String())

		// Add client to tracked connections
		s.clientsMutex.Lock()
		s.clients[conn.RemoteAddr().String()] = conn
		s.clientsMutex.Unlock()

		// Handle the client connection
		go s.handleConnection(conn)
	}
}

// handleConnection handles a client connection
// Implements the Modbus TCP message handling as defined in the specification
// Ref: Modbus_Messaging_Implementation_Guide_V1_0b.pdf, Section 3 (Message Processing)
func (s *TCPServer) handleConnection(conn net.Conn) {
	ctx := context.Background()
	remoteAddr := conn.RemoteAddr().String()
	defer func() {
		// Remove client from tracked connections
		s.clientsMutex.Lock()
		delete(s.clients, remoteAddr)
		s.clientsMutex.Unlock()

		// Close the connection
		conn.Close()
		s.logger.Info(ctx, "Client disconnected: %s", remoteAddr)
	}()

	// Create request timeout for long-running connections
	for {
		// Set a read deadline to prevent hanging forever
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Read the Modbus TCP header (7 bytes)
		// Ref: Modbus_Messaging_Implementation_Guide_V1_0b.pdf, Section 3.1 (MBAP Header)
		// The MBAP header contains:
		// - Transaction Identifier (2 bytes)
		// - Protocol Identifier (2 bytes = 0 for Modbus)
		// - Length (2 bytes, number of following bytes including unit ID)
		// - Unit Identifier (1 byte)
		header := make([]byte, common.TCPHeaderLength)
		_, err := io.ReadFull(conn, header)
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				// Normal client disconnect
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, just continue
				continue
			}
			s.logger.Error(ctx, "Error reading header from %s: %v", remoteAddr, err)
			return
		}

		// Parse MBAP header, using big-endian as per Modbus specification
		// Ref: Modbus_Messaging_Implementation_Guide_V1_0b.pdf, Section 3.1 (MBAP Header)
		// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 4.3 (Data Encoding)
		transactionID := common.TransactionID(binary.BigEndian.Uint16(header[0:2]))
		protocolID := common.ProtocolID(binary.BigEndian.Uint16(header[2:4]))
		length := binary.BigEndian.Uint16(header[4:6])
		unitID := common.UnitID(header[6])

		// Validate protocol ID
		if protocolID != common.TCPProtocolIdentifier {
			s.logger.Error(ctx, "Invalid protocol ID from %s: %d", remoteAddr, protocolID)
			continue
		}

		// Read the PDU (length - 1 bytes, already read unitID)
		dataLength := int(length) - 1
		if dataLength <= 0 {
			s.logger.Error(ctx, "Invalid data length from %s: %d", remoteAddr, length)
			continue
		}

		data := make([]byte, dataLength)
		_, err = io.ReadFull(conn, data)
		if err != nil {
			s.logger.Error(ctx, "Error reading data from %s: %v", remoteAddr, err)
			return
		}

		// Extract function code and PDU data
		// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 5 (Protocol Data Unit)
		// The PDU consists of:
		// - Function Code (1 byte)
		// - Data (variable length, function-specific)
		functionCode := common.FunctionCode(data[0])
		pduData := data[1:]

		// Create a request
		request := transport.NewRequest(unitID, functionCode, pduData)
		request.SetTransactionID(transactionID)

		s.logger.Debug(ctx, "Received request from %s: txID=%d, unit=%d, function=%s",
			remoteAddr, transactionID, unitID, functionCode)

		// Handle the request
		response, err := s.dispatchRequest(ctx, request)
		if err != nil {
			// If it's a Modbus error, create an exception response
			// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 7 (Exception Responses)
			if modbusErr, ok := err.(*common.ModbusError); ok {
				exceptionCode := modbusErr.ExceptionCode
				s.logger.Debug(ctx, "Modbus exception: %s", err.Error())

				// Create an exception response
				// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 7 (Exception Response PDU)
				// Exception responses set the high bit (0x80) in the function code
				exceptionResponse := transport.NewResponse(
					transactionID,
					unitID,
					functionCode|0x80, // Set the high bit for exception response
					[]byte{byte(exceptionCode)},
				)
				s.sendResponse(conn, exceptionResponse)
			} else {
				// For other errors, log and disconnect
				s.logger.Error(ctx, "Error processing request from %s: %v", remoteAddr, err)
				return
			}
			continue
		}

		// Send the response
		s.sendResponse(conn, response)
	}
}

// dispatchRequest dispatches a request to the appropriate handler
// Routes requests to the registered handler for the specified function code
// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 6 (Function Codes)
func (s *TCPServer) dispatchRequest(ctx context.Context, request common.Request) (common.Response, error) {
	// Get the function code
	functionCode := request.GetPDU().FunctionCode

	// Find the handler
	s.mutex.RLock()
	handler, exists := s.handlers[functionCode]
	s.mutex.RUnlock()

	if !exists {
		// Function code not supported, return an exception
		// Ref: Modbus_Application_Protocol_V1_1b3.pdf, Section 7 (Exception Codes)
		// Exception code 0x01 = Illegal Function
		return nil, &common.ModbusError{
			FunctionCode:  functionCode,
			ExceptionCode: common.ExceptionFunctionCodeNotSupported,
		}
	}

	// Call the handler
	return handler(ctx, request)
}

// sendResponse sends a response back to the client
// Encodes the Modbus Application Protocol response and sends it over the TCP connection
// Ref: Modbus_Messaging_Implementation_Guide_V1_0b.pdf, Section 3 (Message Encoding)
func (s *TCPServer) sendResponse(conn net.Conn, response common.Response) {
	ctx := context.Background()
	// Encode the full Modbus TCP message (MBAP Header + PDU)
	// Ref: Modbus_Messaging_Implementation_Guide_V1_0b.pdf, Section 3.1 (MBAP Header)
	data, err := response.Encode()
	if err != nil {
		s.logger.Error(ctx, "Error encoding response: %v", err)
		return
	}

	// Send the encoded response to the client
	_, err = conn.Write(data)
	if err != nil {
		s.logger.Error(ctx, "Error sending response: %v", err)
		return
	}

	s.logger.Debug(ctx, "Sent response: txID=%d, function=%s",
		response.GetTransactionID(), response.GetPDU().FunctionCode)
}
