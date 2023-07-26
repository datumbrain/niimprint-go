package main

import (
	"encoding/binary"
	"fmt"
	"net"
	m "pConv/models"
	"time"
)

type PrinterClient struct {
	Conn      net.Conn
	Packetbuf []byte
}

// InfoEnum is an enumeration for different types of information.
type InfoEnum int

const (
	DENSITY          InfoEnum = 1
	PRINTSPEED       InfoEnum = 2
	LABELTYPE        InfoEnum = 3
	LANGUAGETYPE     InfoEnum = 6
	AUTOSHUTDOWNTIME InfoEnum = 7
	DEVICETYPE       InfoEnum = 8
	SOFTVERSION      InfoEnum = 9
	BATTERY          InfoEnum = 10
	DEVICESERIAL     InfoEnum = 11
	HARDVERSION      InfoEnum = 12
)

// RequestCodeEnum is an enumeration for different request codes.
type RequestCodeEnum int

const (
	GET_INFO          RequestCodeEnum = 64
	GET_RFID          RequestCodeEnum = 26
	HEARTBEAT         RequestCodeEnum = 220
	SET_LABEL_TYPE    RequestCodeEnum = 35
	SET_LABEL_DENSITY RequestCodeEnum = 33
	START_PRINT       RequestCodeEnum = 1
	END_PRINT         RequestCodeEnum = 243
	START_PAGE_PRINT  RequestCodeEnum = 3
	END_PAGE_PRINT    RequestCodeEnum = 227
	ALLOW_PRINT_CLEAR RequestCodeEnum = 32
	SET_DIMENSION     RequestCodeEnum = 19
	SET_QUANTITY      RequestCodeEnum = 21
	GET_PRINT_STATUS  RequestCodeEnum = 163
)

// _packetToInt is a helper function to convert packet data to an integer (big-endian).
func _packetToInt(data []byte) int {
	return int(binary.BigEndian.Uint16(data))
}

// NewPrinterClient creates a new PrinterClient and connects to the given address.
func NewPrinterClient(address string) *PrinterClient {
	conn, err := net.Dial("tcp", address+":1")
	if err != nil {
		return nil
	}
	return &PrinterClient{
		Conn:      conn,
		Packetbuf: make([]byte, 0),
	}
}

// recv receives and deserializes packets from the connection.
func (c *PrinterClient) recv() ([]*m.NiimbotPacket, error) {
	packets := make([]*m.NiimbotPacket, 0)
	buffer := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(buffer)
		if err != nil {
			return nil, err
		}
		c.Packetbuf = append(c.Packetbuf, buffer[:n]...)

		for len(c.Packetbuf) > 4 {
			pktLen := int(c.Packetbuf[3]) + 7
			if len(c.Packetbuf) >= pktLen {
				packet, err := FromBytes(c.Packetbuf[:pktLen])
				if err != nil {
					return nil, err
				}
				packets = append(packets, packet)
				c.Packetbuf = c.Packetbuf[pktLen:]
			} else {
				break
			}
		}
	}
}

// send sends a packet to the connection.
func (c *PrinterClient) send(packet *m.NiimbotPacket) {
	c.Conn.Write(ToBytes(packet))
}

// transceive sends a request packet and waits for a response.
func (c *PrinterClient) transceive(reqCode RequestCodeEnum, data []byte, respOffset byte) (*m.NiimbotPacket, error) {
	respCode := RequestCodeEnum(int(reqCode) + int(respOffset))
	packet := NewNiimbotPacket(byte(reqCode), data)
	c.send(packet)

	for i := 0; i < 6; i++ {
		packets, err := c.recv()
		if err != nil {
			return nil, err
		}
		for _, packet := range packets {
			if packet.Type == 219 {
				return nil, fmt.Errorf("response error")
			} else if packet.Type == 0 {
				return nil, fmt.Errorf("not implemented")
			} else if packet.Type == byte(respCode) {
				return packet, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("response not received")
}

// GetInfo gets the specified information from the printer.
func (c *PrinterClient) GetInfo(key InfoEnum) (interface{}, error) {
	packet, err := c.transceive(GET_INFO, []byte{byte(key)}, byte(key))
	if err != nil {
		return nil, err
	}

	switch key {
	case DEVICESERIAL:
		return fmt.Sprintf("%x", packet.Data), nil
	case SOFTVERSION:
		return _packetToInt(packet.Data) / 100.0, nil
	case HARDVERSION:
		return _packetToInt(packet.Data) / 100.0, nil
	default:
		return _packetToInt(packet.Data), nil
	}
}

// GetRFID gets RFID information from the printer.
func (c *PrinterClient) GetRFID() (map[string]interface{}, error) {
	packet, err := c.transceive(GET_RFID, []byte{0x01}, 0)
	if err != nil {
		return nil, err
	}

	data := packet.Data
	if data[0] == 0 {
		return nil, nil
	}

	idx := 0
	uuid := fmt.Sprintf("%x", data[idx:idx+8])
	idx += 8

	barcodeLen := int(data[idx])
	idx++
	barcode := string(data[idx : idx+barcodeLen])
	idx += barcodeLen

	serialLen := int(data[idx])
	idx++
	serial := string(data[idx : idx+serialLen])
	idx += serialLen

	totalLen := binary.BigEndian.Uint16(data[idx:])
	idx += 2
	usedLen := binary.BigEndian.Uint16(data[idx:])
	idx += 2
	type_ := data[idx]

	return map[string]interface{}{
		"uuid":      uuid,
		"barcode":   barcode,
		"serial":    serial,
		"used_len":  usedLen,
		"total_len": totalLen,
		"type":      type_,
	}, nil
}

// Heartbeat sends a heartbeat request and returns the response.
func (c *PrinterClient) Heartbeat() map[string]interface{} {
	packet, err := c.transceive(HEARTBEAT, []byte{0x01}, 0)
	if err != nil {
		return nil
	}

	data := packet.Data
	heartbeatData := make(map[string]interface{})

	switch len(data) {
	case 20:
		heartbeatData["paperstate"] = int(data[18])
		heartbeatData["rfidreadstate"] = int(data[19])
	case 13:
		heartbeatData["closingstate"] = int(data[9])
		heartbeatData["powerlevel"] = int(data[10])
		heartbeatData["paperstate"] = int(data[11])
		heartbeatData["rfidreadstate"] = int(data[12])
	case 19:
		heartbeatData["closingstate"] = int(data[15])
		heartbeatData["powerlevel"] = int(data[16])
		heartbeatData["paperstate"] = int(data[17])
		heartbeatData["rfidreadstate"] = int(data[18])
	case 10:
		heartbeatData["closingstate"] = int(data[8])
		heartbeatData["powerlevel"] = int(data[9])
		heartbeatData["rfidreadstate"] = int(data[8])
	case 9:
		heartbeatData["closingstate"] = int(data[8])
	}

	return heartbeatData
}

// SetLabelType sets the label type.
func (c *PrinterClient) SetLabelType(n byte) bool {
	assert := func(value bool) {
		if !value {
			panic("assertion failed")
		}
	}
	assert(1 <= n && n <= 3)
	packet, err := c.transceive(SET_LABEL_TYPE, []byte{byte(n)}, 16)
	if err != nil {
		return false
	}
	return packet.Data[0] != 0
}

// SetLabelDensity sets the label density.
func (c *PrinterClient) SetLabelDensity(n byte) bool {
	assert := func(value bool) {
		if !value {
			panic("assertion failed")
		}
	}
	assert(1 <= n && n <= 3)
	packet, err := c.transceive(SET_LABEL_DENSITY, []byte{byte(n)}, 16)
	if err != nil {
		return false
	}
	return packet.Data[0] != 0
}

// StartPrint sends a start print request and returns the response.
func (c *PrinterClient) StartPrint() bool {
	packet, err := c.transceive(START_PRINT, []byte{0x01}, 0)
	if err != nil {
		return false
	}
	return packet.Data[0] != 0
}

// EndPrint sends an end print request and returns the response.
func (c *PrinterClient) EndPrint() bool {
	packet, err := c.transceive(END_PRINT, []byte{0x01}, 0)
	if err != nil {
		return false
	}
	return packet.Data[0] != 0
}

// StartPagePrint sends a start page print request and returns the response.
func (c *PrinterClient) StartPagePrint() bool {
	packet, err := c.transceive(START_PAGE_PRINT, []byte{0x01}, 0)
	if err != nil {
		return false
	}
	return packet.Data[0] != 0
}

// EndPagePrint sends an end page print request and returns the response.
func (c *PrinterClient) EndPagePrint() bool {
	packet, err := c.transceive(END_PAGE_PRINT, []byte{0x01}, 0)
	if err != nil {
		return false
	}
	return packet.Data[0] != 0
}

// AllowPrintClear sends an allow print clear request and returns the response.
func (c *PrinterClient) AllowPrintClear() bool {
	var packet, err = c.transceive(ALLOW_PRINT_CLEAR, []byte{0x01}, 16)
	if err != nil {
		return false
	}

	// Check if the response is valid and return the result
	if len(packet.Data) > 0 && packet.Data[0] == 0x01 {
		return true
	}
	return false
}

func (c *PrinterClient) SetDimension(width, height int) bool {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data, uint16(width))
	binary.BigEndian.PutUint16(data[2:], uint16(height))

	packet, err := c.transceive(SET_DIMENSION, data, 16)
	if err != nil {
		return false
	}

	// Check if the response is valid and return the result
	if len(packet.Data) > 0 && packet.Data[0] == 0x01 {
		return true
	}
	return false
}

func (c *PrinterClient) SetQuantity(quantity uint16) bool {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, quantity)

	packet, err := c.transceive(SET_QUANTITY, data, 16)
	if err != nil {
		return false
	}

	// Check if the response is valid and return the result
	if len(packet.Data) > 0 && packet.Data[0] == 0x01 {
		return true
	}
	return false
}

func (c *PrinterClient) GetPrintStatus() m.PrintStatus {
	packet, err := c.transceive(GET_PRINT_STATUS, []byte{0x01}, 16)
	if err != nil {
		// Handle the error, e.g., return a default or error status
		return m.PrintStatus{}
	}

	// Ensure that the packet data contains enough bytes to extract the print status.
	if len(packet.Data) < 4 {
		// Handle the case where the packet data is too short to parse.
		// For example, return a default or error status.
		return m.PrintStatus{}
	}

	// Extract the print status information from the packet data.
	pageNumber := binary.BigEndian.Uint16(packet.Data)
	progress1 := packet.Data[2]
	progress2 := packet.Data[3]

	// Create and return the PrintStatus struct.
	return m.PrintStatus{
		Page:      pageNumber,
		Progress1: progress1,
		Progress2: progress2,
	}
}
