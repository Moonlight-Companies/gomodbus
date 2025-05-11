# gomodbus

A modern, thread-safe, and type-safe Modbus TCP client/server library for Go.

## Features

- **Type-Safe**: Uses semantic type aliases for better code clarity and error prevention
- **Thread-safe**: Multiple goroutines can use the same client/server instance concurrently
- **Async operation**: Non-blocking I/O with proper context support for cancellation and timeouts
- **Comprehensive**: Implements both client and server components of the Modbus TCP protocol
- **Modular design**: Distinct packages for transport, protocol, client, and server components
- **Customizable logging**: Pluggable logger interface allows integration with any logging framework
- **Fluent configuration API**: Easy to use with functional options pattern for configuration
- **Interface-based design**: Clean separation of concerns with well-defined interfaces
- **Memory store implementation**: Built-in memory-based data store for quick server setup
- **IO abstraction**: Uses standard Go io.Reader and io.Writer interfaces for flexibility and testability

## Package Structure

- `common` - Common interfaces, types, constants, and utilities shared across packages
- `transport` - Transport layer for communication
- `protocol` - Modbus protocol encoding/decoding
- `client` - Modbus client implementations
- `server` - Modbus server and data store implementations
- `logging` - Logging implementations

## Installation

```bash
go get github.com/Moonlight-Companies/gomodbus
```

## Client Usage

### Creating a TCP Client

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/Moonlight-Companies/gomodbus/client"
    "github.com/Moonlight-Companies/gomodbus/common"
    "github.com/Moonlight-Companies/gomodbus/logging"
    "github.com/Moonlight-Companies/gomodbus/transport"
)

func main() {
    // Create a logger
    logger := logging.NewLogger(
        logging.WithLevel(common.LevelInfo),
    )

    // Create a client with custom options
    modbusClient := client.NewTCPClient(
        "localhost",
        transport.WithPort(502),
        transport.WithTimeoutOption(5*time.Second),
        transport.WithTransportLogger(logger),
    ).WithOptions(
        client.WithTCPLogger(logger),
        client.WithTCPUnitID(1),
    )

    // Connect to the server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := modbusClient.Connect(ctx)
    if err != nil {
        fmt.Printf("Failed to connect: %v\n", err)
        return
    }
    defer modbusClient.Disconnect(context.Background())

    // Now you can use the client for Modbus operations
}
```

### Reading Registers and Coils

```go
// Read 10 holding registers starting at address 2000
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

registers, err := modbusClient.ReadHoldingRegisters(ctx, common.Address(2000), common.Quantity(10))
if err != nil {
    fmt.Printf("Failed to read holding registers: %v\n", err)
    return
}

fmt.Println("Holding registers:")
for i, value := range registers {
    fmt.Printf("Register %d: %d (0x%04X)\n", i, value, value)
}

// Read coils
coils, err := modbusClient.ReadCoils(ctx, common.Address(1000), common.Quantity(16))
if err != nil {
    fmt.Printf("Failed to read coils: %v\n", err)
    return
}

fmt.Println("Coils:")
for i, value := range coils {
    fmt.Printf("Coil %d: %t\n", i, value)
}
```

### Writing Registers and Coils

```go
// Write a single register
ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

err = modbusClient.WriteSingleRegister(ctx, common.Address(2000), common.RegisterValue(42))
if err != nil {
    fmt.Printf("Failed to write single register: %v\n", err)
    return
}

// Write multiple registers
ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

values := []common.RegisterValue{1, 2, 3, 4, 5}
err = modbusClient.WriteMultipleRegisters(ctx, common.Address(2000), values)
if err != nil {
    fmt.Printf("Failed to write multiple registers: %v\n", err)
    return
}

// Write a single coil
err = modbusClient.WriteSingleCoil(ctx, common.Address(1000), common.CoilValue(true))
if err != nil {
    fmt.Printf("Failed to write single coil: %v\n", err)
    return
}

// Write multiple coils
coilValues := []common.CoilValue{true, false, true, false, true}
err = modbusClient.WriteMultipleCoils(ctx, common.Address(1000), coilValues)
if err != nil {
    fmt.Printf("Failed to write multiple coils: %v\n", err)
    return
}
```

### Combined Read/Write Operation

```go
// Read from one address and write to another in a single transaction
ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

readAddress := common.Address(2000)
readQuantity := common.Quantity(5)
writeAddress := common.Address(2100)
writeValues := []common.RegisterValue{10, 20, 30, 40, 50}

readValues, err := modbusClient.ReadWriteMultipleRegisters(
    ctx, readAddress, readQuantity, writeAddress, writeValues)
if err != nil {
    fmt.Printf("Failed to execute read/write operation: %v\n", err)
    return
}

fmt.Println("Read values:", readValues)
```

### Reading Device Identification

```go
// Read basic device identification (objects 0x00-0x02)
ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

deviceID, err := modbusClient.ReadDeviceIdentification(ctx, common.ReadDeviceIDBasicStream, common.DeviceIDObjectCode(0))
if err != nil {
    // Check if the error is because the device doesn't support this function
    if common.IsFunctionNotSupportedError(err) {
        fmt.Println("Device identification is not supported by this device")
    } else {
        fmt.Printf("Failed to read device identification: %v\n", err)
    }
    return
}

// Access basic device information (mandatory objects)
fmt.Printf("Vendor Name: %s\n", deviceID.GetVendorName())
fmt.Printf("Product Code: %s\n", deviceID.GetProductCode())
fmt.Printf("Revision: %s\n", deviceID.GetRevision())

// Read regular device identification (includes objects 0x00-0x06)
regularID, err := modbusClient.ReadDeviceIdentification(ctx, common.ReadDeviceIDRegularStream, common.DeviceIDObjectCode(0))
if err == nil {
    fmt.Printf("Product Name: %s\n", regularID.GetProductName())
    fmt.Printf("Model Name: %s\n", regularID.GetModelName())
}

// Read a specific object (vendor URL)
specificID, err := modbusClient.ReadDeviceIdentification(ctx, common.ReadDeviceIDSpecificObject, common.DeviceIDVendorURL)
if err != nil {
    fmt.Printf("Failed to read vendor URL: %v\n", err)
    return
}

if obj := specificID.GetObject(common.DeviceIDVendorURL); obj != nil {
    fmt.Printf("Vendor URL: %s\n", obj.Value)
}
```

### Concurrent Operations

The library supports concurrent operations from multiple goroutines:

```go
func main() {
    // ... setup client as above ...

    var wg sync.WaitGroup
    numRequests := 10
    wg.Add(numRequests)

    for i := 0; i < numRequests; i++ {
        go func(id int) {
            defer wg.Done()

            ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
            defer cancel()

            // Each goroutine reads different registers
            address := common.Address(1000 + id*10)
            quantity := common.Quantity(10)

            values, err := modbusClient.ReadHoldingRegisters(ctx, address, quantity)
            if err != nil {
                fmt.Printf("Request %d failed: %v\n", id, err)
                return
            }

            fmt.Printf("Request %d: Read %d values\n", id, len(values))
        }(i)
    }

    wg.Wait()
}
```

## Server Usage

### Creating a Modbus TCP Server

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/Moonlight-Companies/gomodbus/common"
    "github.com/Moonlight-Companies/gomodbus/logging"
    "github.com/Moonlight-Companies/gomodbus/server"
)

func main() {
    // Create a logger
    logger := logging.NewLogger(
        logging.WithLevel(common.LevelInfo),
    )

    // Create a memory-based data store
    store := server.NewMemoryStore()

    // Pre-populate some data
    store.SetCoil(common.Address(1000), true)
    store.SetCoil(common.Address(1001), false)

    store.SetHoldingRegister(common.Address(2000), 0x1234)
    store.SetHoldingRegister(common.Address(2001), 0x5678)

    // Create a TCP server
    modbusServer := server.NewTCPServer(
        "0.0.0.0", // Listen on all interfaces
        server.WithServerPort(502), // Standard Modbus port
        server.WithServerLogger(logger),
        server.WithServerDataStore(store),
    )

    // Create a context that can be cancelled
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle graceful shutdown
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

    // Start the server in a goroutine
    go func() {
        err := modbusServer.Start(ctx)
        if err != nil && err != context.Canceled {
            fmt.Printf("Server error: %v\n", err)
        }
    }()

    fmt.Println("Modbus server started. Press Ctrl+C to stop.")

    // Wait for termination signal
    <-sig
    fmt.Println("Shutting down server...")

    // Create a context with timeout for shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()

    // Stop the server
    err := modbusServer.Stop(shutdownCtx)
    if err != nil {
        fmt.Printf("Error stopping server: %v\n", err)
    }
}
```

## Advanced Configuration

### Customizing the Logger

```go
// Create a custom logger with specific configuration
logger := logging.NewLogger(
    logging.WithLevel(common.LevelDebug),
    logging.WithWriter(os.Stderr),
)

// Use logger with client
client := client.NewTCPClient("localhost").WithOptions(
    client.WithTCPLogger(logger),
)

// Use logger with server
server := server.NewTCPServer(
    "0.0.0.0",
    server.WithServerLogger(logger),
)
```

### Customizing Timeouts

```go
// Set specific timeouts for the transport layer
client := client.NewTCPClient(
    "localhost",
    transport.WithTimeoutOption(10*time.Second),  // Default transaction timeout
)

// Use context for operation-specific timeouts
ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer cancel()

// This operation will time out after 500ms
values, err := client.ReadHoldingRegisters(ctx, common.Address(0), common.Quantity(10))
```

### Error Handling

The library provides helper functions for checking specific Modbus errors:

```go
// Execute a Modbus function
result, err := client.ReadDeviceIdentification(ctx, common.ReadDeviceIDBasicStream, common.DeviceIDObjectCode(0))
if err != nil {
    // Check if the function is not supported
    if common.IsFunctionNotSupportedError(err) {
        fmt.Println("This device doesn't support device identification")
    }

    // Check for any specific exception
    if common.IsExceptionError(err, common.ExceptionDataAddressNotAvailable) {
        fmt.Println("The requested address is not available")
    }

    // Check if it's any kind of Modbus exception
    if common.IsModbusError(err) {
        fmt.Println("A Modbus exception occurred")
    }

    return
}
```

### Testing with Dynamic Ports

The library provides a utility function to find a free port for testing:

```go
// Find a free port for testing
port, err := common.FindFreePortTCP()
if err != nil {
    t.Fatalf("Failed to find free port: %v", err)
}

// Use the port for server and client
server := server.NewTCPServer(
    "127.0.0.1",
    server.WithServerPort(port),
)

client := client.NewTCPClient(
    "127.0.0.1",
    transport.WithPort(port),
)
```

## Supported Modbus Functions

- Read Coils (0x01)
- Read Discrete Inputs (0x02)
- Read Holding Registers (0x03)
- Read Input Registers (0x04)
- Write Single Coil (0x05)
- Write Single Register (0x06)
- Read Exception Status (0x07)
- Write Multiple Coils (0x0F)
- Write Multiple Registers (0x10)
- Read/Write Multiple Registers (0x17)
- Read Device Identification (0x2B / 0x0E)

### Read Exception Status Example

```go
// Read the exception status from the device
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

status, err := modbusClient.ReadExceptionStatus(ctx)
if err != nil {
    fmt.Printf("Failed to read exception status: %v\n", err)
    return
}

// The ExceptionStatus type provides useful string representation
fmt.Printf("Exception Status: %s\n", status)

// Check if specific bits are set
if status&0x01 != 0 {
    fmt.Println("Exception 0 is set")
}
if status&0x02 != 0 {
    fmt.Println("Exception 1 is set")
}

// Status can also be used as a bit array
for i := 0; i < 8; i++ {
    if status&(1<<i) != 0 {
        fmt.Printf("Exception bit %d is set\n", i)
    }
}
```

## Type-Safe Design

The library uses semantic type aliases to improve code clarity and prevent errors:

```go
// Address represents a Modbus address (coil, register, etc.)
type Address uint16

// Quantity represents the number of coils or registers to read/write
type Quantity uint16

// CoilValue alias represents a coil value
type CoilValue = bool

// RegisterValue alias represents a holding register value
type RegisterValue = uint16

// InputRegisterValue alias represents an input register value
type InputRegisterValue = uint16

// ExceptionStatus represents the exception status returned by a device
type ExceptionStatus byte
```

This allows for more expressive and self-documenting code:

```go
// More expressive and clear
client.ReadHoldingRegisters(ctx, common.Address(2000), common.Quantity(10))

// Versus the less clear version
client.ReadHoldingRegisters(ctx, 2000, 10)
```

## Architecture

The library is designed with clear separation of concerns:

- **Client Interface**: Defines the API for all Modbus clients
- **Server Interface**: Defines the API for Modbus servers
- **DataStore Interface**: Abstraction for data storage in servers
- **Transport**: Handles communication (TCP)
- **Protocol Handler**: Encodes/decodes Modbus PDUs
- **Request/Response**: Represents Modbus messages
- **Transaction Handling**: Manages concurrent requests and responses
- **Logging**: Provides customizable logging capabilities

This architecture makes the library highly extensible, testable, and maintainable.

## License

MIT