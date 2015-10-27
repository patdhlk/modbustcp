package modbustcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	FunctionReadCoil                  = 1
	FunctionReadDiscreteInputs        = 2
	FunctionReadHoldingRegister       = 3
	FunctionReadInputRegister         = 4
	FunctionWriteSingleCoil           = 5
	FunctionWriteSingleRegister       = 6
	FunctionWriteMultipleCoils        = 15
	FunctionWriteMultipleRegister     = 16
	FunctionReadWriteMultipleRegister = 23
)

const (
	ExcIllegalFunction         = 1
	ExcIllegalDataAdr          = 2
	ExcIllegalDataVal          = 3
	ExcSlaveDeviceFailure      = 4
	ExcAcknowledge             = 5
	ExcSlaveIsBusy             = 6
	ExcGatePathUnavailable     = 10
	ExcExceptionNotConnected   = 253
	ExcExceptionConnectionLost = 254
	ExcExceptionTimeout        = 255
	ExcExceptionOffset         = 128
	ExcSendFailt               = 100
)

const (
	TcpProtocolIdentifier uint16 = 0x0000

	// Modbus Application Protocol
	HeaderSize = 7
	MaxLength  = 260
	// Default TCP timeout is not set
	TimeoutMillis = 5000
)

var (
	// ErrorIllegalFunction The function code received
	// in the query is not an allowable action for the slave.
	// This may be because the function code is only applicable to newer devices,
	// and was not implemented in the unit selected.
	// It could also indicate that the slave is in the wrong
	// state to process a request of this type, for example because
	// it is unconfigured and is being asked to return register values.
	// If a Poll Program Complete command was issued, this code indicates that
	// no program function preceded it.
	ErrorIllegalFunction = errors.New("Illegal Function")

	// ErrorIllegalDataAddress The data address received in the query
	// is not an allowable address for the slave. More specifically,
	// the combination of reference number and transfer length is invalid.
	// For a controller with 100 registers, a request with offset 96 and length 4
	// would succeed, a request with offset 96 and length 5 will generate exception 02.
	ErrorIllegalDataAddress = errors.New("Illegal Data Address")

	// ErrorIllegalDataValue A value contained in the query data field
	// is not an allowable value for the slave.
	// This indicates a fault in the structure of remainder of a complex request,
	// such as that the implied length is incorrect. It specifically does NOT
	// mean that a data item submitted for storage in a register has a value
	// outside the expectation of the application program, since the MODBUS
	// protocol is unaware of the
	// significance of any particular value of any particular register.
	ErrorIllegalDataValue = errors.New("Illegal Data Value")

	// ErrorSlaveDeviceFailure An unrecoverable error
	// occurred while the slave was attempting to perform the requested action.
	ErrorSlaveDeviceFailure = errors.New("Slave Device Failure")

	// Specialized use in conjunction with programming commands.
	// The slave has accepted the request and is processing it,
	// but a long duration of time will be required to do so.
	// This response is returned to prevent a timeout error from
	// occurring in the master. The master can next issue a Poll
	// Program Complete message to determine if processing is completed.
	ErrorAcknowledge = errors.New("Acknowledge")

	// Specialized use in conjunction with programming commands.
	// The slave is engaged in processing a long-duration program
	// command.  The master should retransmit the message later
	// when the slave is free..
	ErrorSlaveIsBusy = errors.New("The Slave is Busy")

	// Specialized use in conjunction with gateways,
	// indicates that the gateway was unable to allocate an
	// internal communication path from the input port to the
	// output port for processing the request. Usually means
	// the gateway is misconfigured or overloaded.
	ErrorGatewayPathUnavailable = errors.New("The gateway path is unavailable")

	//handle unknown error code
	ErrorUnknown = errors.New("unknown error occured")
)

type ModbusTcpClient struct {
	IpAddress     string
	Port          int
	Timeout       time.Duration
	SlaveId       byte
	TransactionId uint16
	Logger        *log.Logger

	Conn net.Conn
}

type Pdu struct {
	FunctionCode byte
	Data         []byte
}

func NewModbusTcpClient(ipAddress string, port int) *ModbusTcpClient {
	return &ModbusTcpClient{
		IpAddress: ipAddress,
		Port:      port,
	}
}

func (c *ModbusTcpClient) Connect() error {
	// Timeout must be specified
	if c.Timeout <= 0 {
		c.Timeout = TimeoutMillis * time.Millisecond
	}
	dialer := net.Dialer{Timeout: c.Timeout}
	conn, err := dialer.Dial("tcp", c.IpAddress)
	c.Conn = conn
	return err
}

// Closes the connection
func (c *ModbusTcpClient) Disconnect() error {
	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil {
			return err
		}
		c.Conn = nil
	}
	return nil
}

func (c *ModbusTcpClient) Encode(pdu *Pdu) ([]byte, error) {
	adu := make([]byte, HeaderSize+1+len(pdu.Data))

	// Transaction identifier
	c.TransactionId++

	binary.BigEndian.PutUint16(adu, c.TransactionId)
	// Protocol identifier
	binary.BigEndian.PutUint16(adu[2:], TcpProtocolIdentifier)

	length := uint16(1 + 1 + len(pdu.Data))
	binary.BigEndian.PutUint16(adu[4:], length)

	adu[6] = c.SlaveId

	// PDU
	adu[HeaderSize] = pdu.FunctionCode
	copy(adu[HeaderSize+1:], pdu.Data)
	return adu, nil
}

func (c *ModbusTcpClient) Decode(adu []byte) (*Pdu, error) {
	// Read length value in the header
	length := binary.BigEndian.Uint16(adu[4:])
	pduLength := len(adu) - HeaderSize

	if pduLength <= 0 || pduLength != int(length-1) {
		err := fmt.Errorf("modbus: length in response '%v' does not match pdu data length '%v'", length-1, pduLength)
		return nil, err
	}
	pdu := &Pdu{}
	// The first byte after header is function code
	pdu.FunctionCode = adu[HeaderSize]
	pdu.Data = adu[HeaderSize+1:]
	return pdu, nil
}

func (c *ModbusTcpClient) Verify(aduRequest []byte, aduResponse []byte) error {
	// Transaction id
	responseVal := binary.BigEndian.Uint16(aduResponse)
	requestVal := binary.BigEndian.Uint16(aduRequest)
	if responseVal != requestVal {
		err := fmt.Errorf("modbus: response transaction id '%v' does not match request '%v'", responseVal, requestVal)
		return err
	}
	// Protocol id
	responseVal = binary.BigEndian.Uint16(aduResponse[2:])
	requestVal = binary.BigEndian.Uint16(aduRequest[2:])
	if responseVal != requestVal {
		err := fmt.Errorf("modbus: response protocol id '%v' does not match request '%v'", responseVal, requestVal)
		return err
	}
	// Unit id (1 byte)
	if aduResponse[6] != aduRequest[6] {
		err := fmt.Errorf("modbus: response unit id '%v' does not match request '%v'", aduResponse[6], aduRequest[6])
		return err
	}
	return nil
}

func (c *ModbusTcpClient) ReadDiscreteInputs(startingAddress, quantity uint16) {

}

func (c *ModbusTcpClient) ReadCoils(startingAddress, quantity uint16) {

}

func (c *ModbusTcpClient) ReadHoldingRegisters() {

}

func (c *ModbusTcpClient) ReadInputRegisters() {

}

func (c *ModbusTcpClient) WriteSingleCoil() {

}

func (c *ModbusTcpClient) WriteSingleRegister() {

}

func (c *ModbusTcpClient) WriteMultipleCoils() {

}

func (c *ModbusTcpClient) WriteMultipleRegisters() {

}

func (c *ModbusTcpClient) ReadWriteMultipleRegisters() {

}

func (c *ModbusTcpClient) Send(request []byte) ([]byte, error) {
	var data [MaxLength]byte
	var response []byte
	if c.Conn == nil {
		if err := c.Connect(); err != nil {
			return response, err
		}
		defer c.Disconnect()
	}
	if c.Logger != nil {
		c.Logger.Printf("modbus: sending % x\n", request)
	}
	if err := c.Conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		return response, err
	}
	if _, err := c.Conn.Write(request); err != nil {
		return response, err
	}
	if _, err := io.ReadFull(c.Conn, data[:HeaderSize]); err != nil {
		return response, err
	}
	length := int(binary.BigEndian.Uint16(data[4:]))
	if length <= 0 {
		c.flush(data[:])
		err := fmt.Errorf("modbus: length in response header '%v' must not be zero", length)
		return response, err
	}
	if length > (MaxLength - (HeaderSize - 1)) {
		c.flush(data[:])
		err := fmt.Errorf("modbus: length in response header '%v' must not greater than '%v'", length, MaxLength-HeaderSize+1)
		return response, err
	}
	// Skip unit id
	length += HeaderSize - 1
	if _, err := io.ReadFull(c.Conn, data[HeaderSize:length]); err != nil {
		return response, err
	}
	response = data[:length]

	if c.Logger != nil {
		c.Logger.Printf("modbus: received % x\n", response)
	}
	return response, nil
}

// Gets the correct error for the specified error code
func FailureCodeToError(errorCode int) error {
	switch {
	case errorCode == ExcIllegalFunction:
		return ErrorIllegalFunction
	case errorCode == ExcIllegalDataAdr:
		return ErrorIllegalDataAddress
	case errorCode == ExcIllegalDataVal:
		return ErrorIllegalDataValue
	case errorCode == ExcSlaveDeviceFailure:
		return ErrorSlaveDeviceFailure
	case errorCode == ExcAcknowledge:
		return ErrorAcknowledge
	case errorCode == ExcSlaveIsBusy:
		return ErrorSlaveIsBusy
	case errorCode == ExcGatePathUnavailable:
		return ErrorGatewayPathUnavailable
	default:
		return ErrorUnknown
	}
}

func (c *ModbusTcpClient) flush(b []byte) error {
	if err := c.Conn.SetReadDeadline(time.Now()); err != nil {
		return err
	}
	// Timeout setting will be reset when reading
	if _, err := c.Conn.Read(b); err != nil {
		// Ignore timeout error
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			err = nil
		}
	}
	return nil
}
