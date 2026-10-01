package csa2

func swapNibble(v byte) byte { return (v >> 4) | (v << 4) }

func computeKey(cw [8]byte) [56]byte {
	// Match libdvbcsa_key_schedule_block(): the control word is loaded
	// little-endian, then seven 64-bit key permutations are generated.
	// The masks below are the exact bit permutation represented by the
	// upstream libdvbcsa kperm[8][256] table, reduced to its 64 basis bits.
	var perm = [8][8]uint64{
		{0x0000000000080000, 0x0000000008000000, 0x0080000000000000, 0x0000400000000000, 0x0000000000000002, 0x0000000000008000, 0x0000001000000000, 0x0000000000400000},
		{0x0100000000000000, 0x2000000000000000, 0x0000008000000000, 0x0000000000200000, 0x0040000000000000, 0x0400000000000000, 0x0004000000000000, 0x0000000010000000},
		{0x0000000000000080, 0x0000000020000000, 0x0000080000000000, 0x0000000000000040, 0x0000000200000000, 0x0000000800000000, 0x0000000000100000, 0x0000000000010000},
		{0x0000800000000000, 0x0000000040000000, 0x0000000100000000, 0x8000000000000000, 0x0000000000000400, 0x0000000000000800, 0x0000000000000010, 0x0000004000000000},
		{0x4000000000000000, 0x0000000004000000, 0x00000010000000000, 0x0000000000040000, 0x0000000000001000, 0x0000100000000000, 0x00000002000000000, 0x00002000000000000},
		{0x0000000000800000, 0x0080000000000000, 0x000000200000000, 0x0000000000020000, 0x0000000080000000, 0x0000000000000001, 0x0000000002000000, 0x0000080000000000},
		{0x0000100000000000, 0x0000000000004000, 0x0000000000000004, 0x0000000000002000, 0x0000020000000000, 0x0000200000000000, 0x0000000000000008, 0x1000000000000000},
		{0x0002000000000000, 0x0000000000000100, 0x0000000400000000, 0x0000000000000020, 0x0000000000000200, 0x0000040000000000, 0x0200000000000000, 0x0000000001000000},
	}

	var k [7]uint64
	for i := 0; i < 8; i++ {
		k[6] |= uint64(cw[i]) << uint(8*i)
	}
	for i := 6; i > 0; i-- {
		var next uint64
		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			v := byte(k[i] >> uint(8*byteIndex))
			for bit := 0; bit < 8; bit++ {
				if v&(1<<uint(bit)) != 0 {
					next |= perm[byteIndex][bit]
				}
			}
		}
		k[i-1] = next
	}

	var kk [56]byte
	for i := 0; i < 7; i++ {
		for j := 0; j < 8; j++ {
			kk[i*8+j] = byte(k[i]>>uint(8*j)) ^ byte(i)
		}
	}
	return kk
}

func blockEncrypt(kk *[56]byte, in [8]byte) [8]byte {
 var r [9]byte
 copy(r[1:],in[:])
 for i:=0;i<56;i++ {
  s:=blockSBox[kk[i]^r[8]]
  p:=blockPerm[s]
  next:=r[2]
  r[2]=r[3]^r[1]
  r[3]=r[4]^r[1]
  r[4]=r[5]^r[1]
  r[5]=r[6]
  r[6]=r[7]^p
  r[7]=r[8]
  r[8]=r[1]^s
  r[1]=next
 }
 var out [8]byte
 copy(out[:],r[1:])
 return out
}

func blockDecrypt(kk *[56]byte, in [8]byte) [8]byte {
 var r [9]byte
 copy(r[1:],in[:])
 for i:=55;i>=0;i-- {
  s:=blockSBox[kk[i]^r[7]]
  p:=blockPerm[s]
  next:=r[7]
  r[7]=r[6]^p
  r[6]=r[5]
  r[5]=r[4]^r[8]^s
  r[4]=r[3]^r[8]^s
  r[3]=r[2]^r[8]^s
  r[2]=r[1]
  r[1]=r[8]^s
  r[8]=next
 }
 var out [8]byte
 copy(out[:],r[1:])
 return out
}
