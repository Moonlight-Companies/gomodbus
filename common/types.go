package common

import "fmt"

// TransactionID is a unique identifier for a transaction
type TransactionID uint16

// ProtocolID identifies the protocol used (e.g., Modbus TCP, RTU)
type ProtocolID uint16

// UnitID identifies a specific device on a Modbus network
type UnitID byte

// ExceptionCode represents an exception code in a Modbus response
type ExceptionCode byte

// FunctionCode represents a Modbus function code
type FunctionCode byte

// Address represents a Modbus address (coil, register, etc.)
type Address uint16

// Quantity represents the number of coils or registers to read/write
type Quantity uint16

// CoilValue alias represents a coil value
type CoilValue = bool

// DiscreteInputValue alias represents a discrete input value
type DiscreteInputValue = bool

// RegisterValue alias represents a holding register value
type RegisterValue = uint16

// InputRegisterValue alias represents an input register value
type InputRegisterValue = uint16

// Function codes as defined by the Modbus specification
const (
	// Standard function codes
	FuncReadCoils                  FunctionCode = 0x01
	FuncReadDiscreteInputs         FunctionCode = 0x02
	FuncReadHoldingRegisters       FunctionCode = 0x03
	FuncReadInputRegisters         FunctionCode = 0x04
	FuncWriteSingleCoil            FunctionCode = 0x05
	FuncWriteSingleRegister        FunctionCode = 0x06
	FuncReadExceptionStatus        FunctionCode = 0x07
	FuncWriteMultipleCoils         FunctionCode = 0x0F
	FuncWriteMultipleRegisters     FunctionCode = 0x10
	FuncReadWriteMultipleRegisters FunctionCode = 0x17

	// Exception codes
	ExceptionFunctionCodeNotSupported ExceptionCode = 0x01
	ExceptionDataAddressNotAvailable  ExceptionCode = 0x02
	ExceptionInvalidDataValue         ExceptionCode = 0x03
	ExceptionServerDeviceFailure      ExceptionCode = 0x04
	ExceptionAcknowledge              ExceptionCode = 0x05
	ExceptionServerDeviceBusy         ExceptionCode = 0x06
	ExceptionMemoryParityError        ExceptionCode = 0x08
	ExceptionGatewayPathUnavailable   ExceptionCode = 0x0A
	ExceptionGatewayTargetNoResponse  ExceptionCode = 0x0B
)

// String returns the string representation of a FunctionCode
func (f FunctionCode) String() string {
	switch f {
	case FuncReadCoils:
		return "ReadCoils"
	case FuncReadDiscreteInputs:
		return "ReadDiscreteInputs"
	case FuncReadHoldingRegisters:
		return "ReadHoldingRegisters"
	case FuncReadInputRegisters:
		return "ReadInputRegisters"
	case FuncWriteSingleCoil:
		return "WriteSingleCoil"
	case FuncWriteSingleRegister:
		return "WriteSingleRegister"
	case FuncReadExceptionStatus:
		return "ReadExceptionStatus"
	case FuncWriteMultipleCoils:
		return "WriteMultipleCoils"
	case FuncWriteMultipleRegisters:
		return "WriteMultipleRegisters"
	case FuncReadWriteMultipleRegisters:
		return "ReadWriteMultipleRegisters"
	default:
		// If it's an exception response
		if IsException(byte(f)) {
			original := GetOriginalFunctionCode(byte(f))
			return fmt.Sprintf("Exception(%s)", FunctionCode(original).String())
		}
		return fmt.Sprintf("Unknown(0x%02X)", byte(f))
	}
}

func (e ExceptionCode) String() string {
	switch e {
	case ExceptionFunctionCodeNotSupported:
		return "FunctionCodeNotSupported"
	case ExceptionDataAddressNotAvailable:
		return "DataAddressNotAvailable"
	case ExceptionInvalidDataValue:
		return "InvalidDataValue"
	case ExceptionServerDeviceFailure:
		return "ServerDeviceFailure"
	case ExceptionAcknowledge:
		return "Acknowledge"
	case ExceptionServerDeviceBusy:
		return "ServerDeviceBusy"
	case ExceptionMemoryParityError:
		return "MemoryParityError"
	case ExceptionGatewayPathUnavailable:
		return "GatewayPathUnavailable"
	case ExceptionGatewayTargetNoResponse:
		return "GatewayTargetNoResponse"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", byte(e))
	}
}

// Protocol-specific constants
const (
	// Modbus TCP
	TCPHeaderLength = 7   // Transaction ID (2) + Protocol ID (2) + Length (2) + Unit ID (1)
	MaxPDULength    = 253 // Maximum PDU length
	MaxADULength    = 260 // Maximum ADU length (TCP with header)
	DefaultTCPPort  = 502 // Default Modbus TCP port

	// Data sizes
	BytesPerCoil          = 1
	BytesPerDiscreteInput = 1
	BytesPerRegister      = 2
	BytesPerInputRegister = 2

	// Modbus limits
	MaxCoilCount     = 2000 // Maximum number of coils in a single request
	MaxRegisterCount = 125  // Maximum number of registers in a single request

	// Modbus protocol constants
	CoilOnU16  = 0xFF00
	CoilOffU16 = 0x0000
)

// TCPProtocolIdentifier is the standard identifier for Modbus TCP
var TCPProtocolIdentifier = ProtocolID(0)

// ExceptionBit is the bit that is set in the function code to indicate an exception response
const ExceptionBit byte = 0x80

// IsException checks if a function code represents an exception
func IsException(functionCode byte) bool {
	return (functionCode & ExceptionBit) != 0
}

// IsFunctionException checks if a FunctionCode represents an exception
func IsFunctionException(functionCode FunctionCode) bool {
	return IsException(byte(functionCode))
}

// GetOriginalFunctionCode extracts the original function code from an exception
func GetOriginalFunctionCode(exceptionCode byte) byte {
	return exceptionCode & ^ExceptionBit
}

// GetOriginalFunction extracts the original FunctionCode from an exception
func GetOriginalFunction(exceptionCode FunctionCode) FunctionCode {
	return FunctionCode(GetOriginalFunctionCode(byte(exceptionCode)))
}
