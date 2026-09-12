package mpegts

// MPEG-2 section CRC-32, polynomial 0x04C11DB7, MSB first.
func CRC32MPEG(data []byte) uint32 {
	crc := uint32(0xffffffff)
	for _, b := range data {
		crc ^= uint32(b) << 24
		for i := 0; i < 8; i++ {
			if crc&0x80000000 != 0 { crc = (crc << 1) ^ 0x04c11db7 } else { crc <<= 1 }
		}
	}
	return crc
}

func validCRC(data []byte) bool { return len(data) >= 4 && CRC32MPEG(data) == 0 }
