package common

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNotConnected          = errors.New("client not connected")
	ErrAlreadyConnected      = errors.New("client already connected")
	ErrInvalidQuantity       = errors.New("invalid quantity")
	ErrInvalidAddress        = errors.New("invalid address")
	ErrInvalidResponseLength = errors.New("invalid response length")
	ErrInvalidCRC            = errors.New("invalid CRC")
	ErrInvalidFunction       = errors.New("invalid function code")
	ErrInvalidValue          = errors.New("invalid value")
	ErrInvalidResponseFormat = errors.New("invalid response format")
	ErrTimeout               = errors.New("timeout")
	ErrContextCanceled       = errors.New("context canceled")
	ErrInvalidProtocolHeader = errors.New("invalid protocol header")
	ErrTooManyRegisters      = errors.New("too many registers requested")
	ErrTooManyCoils          = errors.New("too many coils requested")
	ErrEmptyResponse         = errors.New("empty response")
	ErrResponseTooLarge      = errors.New("response too large")
	ErrRequestTooLarge       = errors.New("request too large")
	ErrTransactionTimeout    = errors.New("transaction timeout")
	ErrTransportClosing      = errors.New("transport closing")
	ErrServerDeviceFailure   = errors.New("server device failure")
	ErrNoResponse            = errors.New("no response from server")
)

// ModbusError represents an error from a Modbus exception response
type ModbusError struct {
	FunctionCode  FunctionCode
	ExceptionCode ExceptionCode
}

// Error implements the error interface
func (e *ModbusError) Error() string {
	return fmt.Sprintf("modbus: exception response: function: %s, exception code: %#x (%s)",
		e.FunctionCode, e.ExceptionCode, GetExceptionString(e.ExceptionCode))
}

// IsModbusError checks if an error is a ModbusError
func IsModbusError(err error) bool {
	_, ok := err.(*ModbusError)
	return ok
}

// IsExceptionError checks if an error is a specific Modbus exception
func IsExceptionError(err error, exceptionCode ExceptionCode) bool {
	if modbusErr, ok := err.(*ModbusError); ok {
		return modbusErr.ExceptionCode == exceptionCode
	}
	return false
}

// IsFunctionNotSupportedError checks if an error is due to a function not being supported
func IsFunctionNotSupportedError(err error) bool {
	return IsExceptionError(err, ExceptionFunctionCodeNotSupported)
}

// NewModbusError creates a new ModbusError
func NewModbusError(functionCode FunctionCode, exceptionCode ExceptionCode) *ModbusError {
	return &ModbusError{
		FunctionCode:  functionCode,
		ExceptionCode: exceptionCode,
	}
}

// GetExceptionString returns a human-readable description of an exception code
func GetExceptionString(exceptionCode ExceptionCode) string {
	switch exceptionCode {
	case ExceptionFunctionCodeNotSupported:
		return "function code not supported"
	case ExceptionDataAddressNotAvailable:
		return "data address not available"
	case ExceptionInvalidDataValue:
		return "invalid data value"
	case ExceptionServerDeviceFailure:
		return "server device failure"
	case ExceptionAcknowledge:
		return "acknowledge"
	case ExceptionServerDeviceBusy:
		return "server device busy"
	case ExceptionMemoryParityError:
		return "memory parity error"
	case ExceptionGatewayPathUnavailable:
		return "gateway path unavailable"
	case ExceptionGatewayTargetNoResponse:
		return "gateway target no response"
	default:
		return fmt.Sprintf("unknown exception code: %#x", exceptionCode)
	}
}
