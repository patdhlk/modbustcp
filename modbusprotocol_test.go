package modbustcp

import (
	"testing"
)

func TestCRC(t *testing.T) {
	var crc CRC
	crc.Reset()
	crc.PushBytes([]byte{0x02, 0x07})

	if 0x4112 != crc.Value() {
		t.Fatalf("crc expected %v, actual %v", 0x4112, crc.Value())
	}
}
