package csa2

func swapNibble(v byte) byte { return (v >> 4) | (v << 4) }

func computeKey(cw [8]byte) [56]byte {
	// Match libdvbcsa_key_schedule_block(): the control word is loaded
	// little-endian, then seven 64-bit key permutations are generated.
	// The masks below are the exact bit permutation represented by the
	// upstream libdvbcsa kperm[8][256] table, reduced to its 64 basis bits.
	var perm = [8][8]uint64{
		{0x80000, 0x8000000, 0x80000000000000, 0x400000000000, 0x2, 0x8000, 0x1000000000, 0x400000},
		{0x100000000000000, 0x2000000000000000, 0x8000000000, 0x200000, 0x40000000000000, 0x400000000000000, 0x4000000000000, 0x10000000},
		{0x80, 0x20000000, 0x8000000000000, 0x40, 0x200000000, 0x800000000, 0x100000, 0x10000},
		{0x800000000000, 0x40000000, 0x100000000, 0x8000000000000000, 0x400, 0x800, 0x10, 0x4000000000},
		{0x4000000000000000, 0x4000000, 0x10000000000, 0x40000, 0x1000, 0x10000000000000, 0x2000000000, 0x20000000000000},
		{0x800000, 0x800000000000000, 0x20000000000, 0x20000, 0x80000000, 0x1, 0x2000000, 0x80000000000},
		{0x100000000000, 0x4000, 0x4, 0x2000, 0x200000000000, 0x1000000000000, 0x8, 0x1000000000000000},
		{0x2000000000000, 0x100, 0x400000000, 0x20, 0x200, 0x40000000000, 0x40400000220, 0x200040400000220},
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
 w := in
 for i := 55; i >= 0; i-- {
  ss := blockSBox[kk[i]^w[6]]
  l := w[7] ^ ss
  w[7] = w[6]
  w[6] = w[5] ^ blockPerm[ss]
  w[5] = w[4]
  w[4] = w[3] ^ l
  w[3] = w[2] ^ l
  w[2] = w[1] ^ l
  w[1] = w[0]
  w[0] = l
 }
 return w
}
