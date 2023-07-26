package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	m "pConv/models"
)

const chunkSize = 12

func countBitsOfBytes(data []byte) int {
	// Count the number of bits set to 1 in the bytes
	n := binary.BigEndian.Uint32(data)
	n = (n & 0x55555555) + ((n & 0xAAAAAAAA) >> 1)
	n = (n & 0x33333333) + ((n & 0xCCCCCCCC) >> 2)
	n = (n & 0x0F0F0F0F) + ((n & 0xF0F0F0F0) >> 4)
	n = (n & 0x00FF00FF) + ((n & 0xFF00FF00) >> 8)
	n = (n & 0x0000FFFF) + ((n & 0xFFFF0000) >> 16)
	return int(n)
}

func naiveEncoder(img image.Image) []*m.NiimbotPacket {
	bounds := img.Bounds()
	packets := make([]*m.NiimbotPacket, 0, bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		imgData := make([]byte, 0, chunkSize)

		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Convert image color to binary value (0 or 1)
			pixel := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			binaryPixel := byte(1)
			if pixel.Y > 128 {
				binaryPixel = 0
			}
			imgData = append(imgData, binaryPixel)
		}

		counts := make([]byte, 3)
		for i := 0; i < 3; i++ {
			// Count bits of bytes in groups of 4 bytes (12 bits each)
			counts[i] = byte(countBitsOfBytes(imgData[i*4 : (i+1)*4]))
		}

		header := new(bytes.Buffer)
		binary.Write(header, binary.BigEndian, uint16(y))
		header.Write(counts)
		header.WriteByte(1)

		// Create a new NiimbotPacket for each iteration and append it to packets
		pkt := &m.NiimbotPacket{
			Type: 0x85,
			Data: append(header.Bytes(), imgData...),
		}
		packets = append(packets, pkt)
	}

	return packets
}
