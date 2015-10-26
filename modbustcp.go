package modbustcp

import (
	"errors"
	"net"
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
	ExcSalveIsBusy             = 6
	ExcGatePathUnavailable     = 10
	ExcExceptionNotConnected   = 253
	ExcExceptionConnectionLost = 254
	ExcExceptionTimeout        = 255
	ExcExceptionOffset         = 128
	ExcSendFailt               = 100
)

type ModbusTcpClient struct {
	IpAddress string
	Port      int
	Timeout   int
	Conn      net.Conn
}

func NewModbusTcpClient(ipAddress string, port int) *ModbusTcpClient {
	return &ModbusTcpClient{
		IpAddress: ipAddress,
		Port:      port,
	}
}

func (c *ModbusTcpClient) Connect() {

}

func (c *ModbusTcpClient) Disconnect() {

}

func (c *ModbusTcpClient) ReadDiscreteInputs(startingAddress, quantity int) ([]bool, error) {
	if startingAddress > 65535 || quantity > 2000 {
		return nil, errors.New("Starting address must be 0-65535, quantity must be 0 - 2000")
	}

}

func (c *ModbusTcpClient) ReadCoils() {

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
