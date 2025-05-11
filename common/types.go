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

// ExceptionStatus represents the return value from ReadExceptionStatus
type ExceptionStatus byte

// ReadDeviceIDCode represents a device identification access type
type ReadDeviceIDCode byte

// DeviceIDObjectCode represents a device identification object code
type DeviceIDObjectCode byte

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
	FuncReadDeviceIdentification   FunctionCode = 0x2B // MEI Transport

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

// MEIType represents a Modbus Encapsulated Interface type
// Used in function code 0x2B (Modbus Encapsulated Interface)
// MEIType indicates which sub-function to execute
type MEIType byte

// MEI Types
const (
	// MEIReadDeviceID is the MEI type for reading device identification (0x0E)
	MEIReadDeviceID MEIType = 0x0E

	// Other MEI types from the specification could be added here:
	// 0x0D - CANopen General Reference Request and Response PDU
	// 0x0A - CUT File Access
	// etc.
)

// Read Device ID codes
const (
	// ReadDeviceIDBasicStream requests basic device identification (stream access for objects 0x00-0x02)
	ReadDeviceIDBasicStream ReadDeviceIDCode = 0x01
	// ReadDeviceIDRegularStream requests regular device identification (stream access through UserApplicationName)
	ReadDeviceIDRegularStream ReadDeviceIDCode = 0x02
	// ReadDeviceIDExtendedStream requests extended device identification (stream access for all objects)
	ReadDeviceIDExtendedStream ReadDeviceIDCode = 0x03
	// ReadDeviceIDSpecificObject requests a specific identification object (individual access)
	ReadDeviceIDSpecificObject ReadDeviceIDCode = 0x04

	// Alias the old names for backwards compatibility
	ReadDeviceIDBasic    = ReadDeviceIDBasicStream
	ReadDeviceIDRegular  = ReadDeviceIDRegularStream
	ReadDeviceIDExtended = ReadDeviceIDExtendedStream
	ReadDeviceIDSpecific = ReadDeviceIDSpecificObject
)

// Device identification object IDs
const (
	// Basic identification objects (mandatory)
	DeviceIDVendorName         DeviceIDObjectCode = 0x00 // VendorName - Mandatory basic object
	DeviceIDProductCode        DeviceIDObjectCode = 0x01 // ProductCode - Mandatory basic object
	DeviceIDMajorMinorRevision DeviceIDObjectCode = 0x02 // Revision - Mandatory basic object

	// Regular identification objects (optional)
	DeviceIDVendorURL   DeviceIDObjectCode = 0x03 // VendorURL - Standard regular object
	DeviceIDProductName DeviceIDObjectCode = 0x04 // ProductName - Standard regular object
	DeviceIDModelName   DeviceIDObjectCode = 0x05 // ModelName - Standard regular object
	DeviceIDUserAppName DeviceIDObjectCode = 0x06 // UserApplicationName - Standard regular object

	// Private objects (vendor-specific)
	// Objects in the range 0x07-0x7F are reserved for future standard objects
	// Objects in the range 0x80-0xFF are vendor-specific extended objects
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
	case FuncReadDeviceIdentification:
		return "ReadDeviceIdentification"
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

// String returns the string representation of a MEIType
func (m MEIType) String() string {
	switch m {
	case MEIReadDeviceID:
		return "ReadDeviceIdentification"
	default:
		return fmt.Sprintf("UnknownMEIType(0x%02X)", byte(m))
	}
}

// String returns the string representation of a ReadDeviceIDCode
func (c ReadDeviceIDCode) String() string {
	switch c {
	case ReadDeviceIDBasicStream:
		return "BasicStream"
	case ReadDeviceIDRegularStream:
		return "RegularStream"
	case ReadDeviceIDExtendedStream:
		return "ExtendedStream"
	case ReadDeviceIDSpecificObject:
		return "SpecificObject"
	default:
		return fmt.Sprintf("UnknownReadDeviceIDCode(0x%02X)", byte(c))
	}
}

// String returns the string representation of a DeviceIDObjectCode
func (c DeviceIDObjectCode) String() string {
	switch c {
	case DeviceIDVendorName:
		return "VendorName"
	case DeviceIDProductCode:
		return "ProductCode"
	case DeviceIDMajorMinorRevision:
		return "MajorMinorRevision"
	case DeviceIDVendorURL:
		return "VendorURL"
	case DeviceIDProductName:
		return "ProductName"
	case DeviceIDModelName:
		return "ModelName"
	case DeviceIDUserAppName:
		return "UserApplicationName"
	default:
		if c >= 0x80 {
			return fmt.Sprintf("ExtendedObject(0x%02X)", byte(c))
		}
		return fmt.Sprintf("UnknownObject(0x%02X)", byte(c))
	}
}

// String returns a string representation of the ExceptionStatus
func (s ExceptionStatus) String() string {
	// Since ExceptionStatus is a bit field (8 coils), show which bits are set
	var bits []int
	for i := 0; i < 8; i++ {
		if (s & (1 << i)) != 0 {
			bits = append(bits, i)
		}
	}

	if len(bits) == 0 {
		return "ExceptionStatus(None)"
	}

	return fmt.Sprintf("ExceptionStatus(Bits: %v, Value: 0x%02X)", bits, byte(s))
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
const TCPProtocolIdentifier = ProtocolID(0)

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
