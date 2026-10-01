package csa2

func swapNibble(v byte) byte { return (v >> 4) | (v << 4) }

func computeKey(cw [8]byte) [57]byte {
 var kk [56]byte
 var kb [8][8]byte
 copy(kb[7][:], cw[:])
 for i:=0;i<7;i++ {
  var bits,newBits [64]byte
  for j:=0;j<8;j++ {
   for k:=0;k<8;k++ {
    bits[j*8+k]=(kb[7-i][j]>>uint(7-k))&1
    newBits[int(keyPerm[j*8+k])-1]=bits[j*8+k]
   }
  }
  for j:=0;j<8;j++ {
   var v byte
   for k:=0;k<8;k++ { v |= newBits[j*8+k]<<uint(7-k) }
   kb[6-i][j]=v
  }
 }
 for i:=0;i<7;i++ {
  for j:=0;j<8;j++ { kk[i*8+j]=kb[i][j]^byte(i) }
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
