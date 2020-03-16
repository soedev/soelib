package des

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/axgle/mahonia"
)

const PowerDesKey = "power"

func EncryStr(aStr string) (encryStr string, err error) {
	powerDes := PowerDes{}
	return powerDes.PowerEncryStr(aStr, PowerDesKey)
}

//DecryStr 封装Delphi Des解密方法
func DecryStr(aStr string) (decryStr string, err error) {
	powerDes := PowerDes{}
	return powerDes.PowerDecryStr(aStr, PowerDesKey)
}

//PowerDes  Delphi  powerDes加密重新
type PowerDes struct {
}

//KeyByte = [5]byte{0, 0, 0, 0, 0}

//BitIP BitIP
var BitIP = [64]byte{57, 49, 41, 33, 25, 17, 9, 1,
	59, 51, 43, 35, 27, 19, 11, 3,
	61, 53, 45, 37, 29, 21, 13, 5,
	63, 55, 47, 39, 31, 23, 15, 7,
	56, 48, 40, 32, 24, 16, 8, 0,
	58, 50, 42, 34, 26, 18, 10, 2,
	60, 52, 44, 36, 28, 20, 12, 4,
	62, 54, 46, 38, 30, 22, 14, 6}

//BitCP BitCP
var BitCP = [64]byte{39, 7, 47, 15, 55, 23, 63, 31,
	38, 6, 46, 14, 54, 22, 62, 30,
	37, 5, 45, 13, 53, 21, 61, 29,
	36, 4, 44, 12, 52, 20, 60, 28,
	35, 3, 43, 11, 51, 19, 59, 27,
	34, 2, 42, 10, 50, 18, 58, 26,
	33, 1, 41, 9, 49, 17, 57, 25,
	32, 0, 40, 8, 48, 16, 56, 24}

//BitExp BitExp
var BitExp = [48]byte{31, 0, 1, 2, 3, 4, 3, 4, 5, 6, 7, 8, 7, 8, 9, 10,
	11, 12, 11, 12, 13, 14, 15, 16, 15, 16, 17, 18, 19, 20, 19, 20,
	21, 22, 23, 24, 23, 24, 25, 26, 27, 28, 27, 28, 29, 30, 31, 0}

//BitPM BitPM
var BitPM = [32]byte{15, 6, 19, 20, 28, 11, 27, 16, 0, 14, 22, 25, 4, 17, 30, 9,
	1, 7, 23, 13, 31, 26, 2, 8, 18, 12, 29, 5, 21, 10, 3, 24}

//BitPMC1 BitPMC1
var BitPMC1 = [56]byte{56, 48, 40, 32, 24, 16, 8,
	0, 57, 49, 41, 33, 25, 17,
	9, 1, 58, 50, 42, 34, 26,
	18, 10, 2, 59, 51, 43, 35,
	62, 54, 46, 38, 30, 22, 14,
	6, 61, 53, 45, 37, 29, 21,
	13, 5, 60, 52, 44, 36, 28,
	20, 12, 4, 27, 19, 11, 3}

//BitPMC2 BitPMC2
var BitPMC2 = [48]byte{13, 16, 10, 23, 0, 4,
	2, 27, 14, 5, 20, 9,
	22, 18, 11, 3, 25, 7,
	15, 6, 26, 19, 12, 1,
	40, 51, 30, 36, 46, 54,
	29, 39, 50, 44, 32, 47,
	43, 48, 38, 55, 33, 52,
	45, 41, 49, 35, 28, 31}

var sBox = [8][64]byte{{14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7,
	0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8,
	4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0,
	15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13},

	{15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10,
		3, 13, 4, 7, 15, 2, 8, 14, 12, 0, 1, 10, 6, 9, 11, 5,
		0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15,
		13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9},

	{10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8,
		13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1,
		13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7,
		1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12},

	{7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15,
		13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9,
		10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4,
		3, 15, 0, 6, 10, 1, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14},

	{2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9,
		14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6,
		4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14,
		11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3},

	{12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11,
		10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8,
		9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6,
		4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13},

	{4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1,
		13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6,
		1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2,
		6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12},

	{13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7,
		1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2,
		7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8,
		2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11}}

const (
	dmEncry = 0
	dmDecry = 1
)

func (p PowerDes) initPermutation(inData *[]byte) {
	var newData = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	var i uint
	for i = 0; i < 64; i++ {

		if ((*inData)[BitIP[i]>>3] & (1 << (7 - (BitIP[i] & 7)))) != 0 {
			newData[i>>3] = newData[i>>3] | (1 << (7 - (i & 7)))
		}
	}

	for i = 0; i < 8; i++ {
		(*inData)[i] = newData[i]
	}
}

func (p PowerDes) conversePermutation(inData *[]byte) {
	var newData = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	var i uint
	for i = 0; i < 64; i++ {

		if ((*inData)[BitCP[i]>>3] & (1 << (7 - (BitCP[i] & 7)))) != 0 {
			newData[i>>3] = newData[i>>3] | (1 << (7 - (i & 7)))
		}
	}

	for i = 0; i < 8; i++ {
		(*inData)[i] = newData[i]
	}
}

func (p PowerDes) expand(inData []byte, outData *[6]byte) {
	var i uint

	for i = 0; i < 6; i++ {
		(*outData)[i] = 0
	}

	for i = 0; i < 48; i++ {

		if ((inData)[BitExp[i]>>3] & (1 << (7 - (BitExp[i] & 7)))) != 0 {
			(*outData)[i>>3] = (*outData)[i>>3] | (1 << (7 - (i & 7)))
		}
	}
}

func (p PowerDes) permutation(inData *[6]byte) {
	var newData = [4]byte{0, 0, 0, 0}
	var i uint
	for i = 0; i < 32; i++ {

		if ((*inData)[BitPM[i]>>3] & (1 << (7 - (BitPM[i] & 7)))) != 0 {
			newData[i>>3] = newData[i>>3] | (1 << (7 - (i & 7)))
		}
	}

	for i = 0; i < 4; i++ {
		(*inData)[i] = newData[i]
	}
}

func (p PowerDes) si(s byte, inByte byte) byte {
	c := (inByte & 0x20) | ((inByte & 0x1e) >> 1) | ((inByte & 0x01) << 4)
	return sBox[s][c] & 0x0f
}

func (p PowerDes) permutationChoose1(inData []byte, outData *[7]byte) {
	var i uint

	for i = 0; i < 7; i++ {
		(*outData)[i] = 0
	}

	for i = 0; i < 56; i++ {

		if ((inData)[BitPMC1[i]>>3] & (1 << (7 - (BitPMC1[i] & 7)))) != 0 {
			(*outData)[i>>3] = (*outData)[i>>3] | (1 << (7 - (i & 7)))
		}
	}
}

func (p PowerDes) permutationChoose2(inData [7]byte, outData *[]byte) {
	var i uint

	for i = 0; i < 6; i++ {
		(*outData)[i] = 0
	}

	for i = 0; i < 48; i++ {

		if ((inData)[BitPMC2[i]>>3] & (1 << (7 - (BitPMC2[i] & 7)))) != 0 {
			(*outData)[i>>3] = (*outData)[i>>3] | (1 << (7 - (i & 7)))
		}
	}
}

func (p PowerDes) cycleMove(inData *[4]byte, bitMove byte) {
	var i byte
	for i = 0; i < bitMove; i++ {
		(*inData)[0] = ((*inData)[0] << 1) | ((*inData)[1] >> 7)
		(*inData)[1] = ((*inData)[1] << 1) | ((*inData)[2] >> 7)
		(*inData)[2] = ((*inData)[2] << 1) | ((*inData)[3] >> 7)
		(*inData)[3] = ((*inData)[3] << 1) | (((*inData)[0] & 0x10) >> 4)
		(*inData)[0] = ((*inData)[0] & 0x0f)
	}
}

func (p PowerDes) makeKey(inKey []byte, outData *[][]byte) {
	var bitDisplace = [16]byte{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}
	var i byte
	var outData56 = [7]byte{0, 0, 0, 0, 0, 0, 0}
	var key28l = [4]byte{0, 0, 0, 0}
	var key28r = [4]byte{0, 0, 0, 0}
	var key56o = [7]byte{0, 0, 0, 0, 0, 0, 0}

	p.permutationChoose1(inKey, &outData56)

	key28l[0] = outData56[0] >> 4
	key28l[1] = (outData56[0] << 4) | (outData56[1] >> 4)
	key28l[2] = (outData56[1] << 4) | (outData56[2] >> 4)
	key28l[3] = (outData56[2] << 4) | (outData56[3] >> 4)
	key28r[0] = outData56[3] & 0x0f
	key28r[1] = outData56[4]
	key28r[2] = outData56[5]
	key28r[3] = outData56[6]

	for i = 0; i < 16; i++ {
		p.cycleMove(&key28l, bitDisplace[i])
		p.cycleMove(&key28r, bitDisplace[i])
		key56o[0] = (key28l[0] << 4) | (key28l[1] >> 4)
		key56o[1] = (key28l[1] << 4) | (key28l[2] >> 4)
		key56o[2] = (key28l[2] << 4) | (key28l[3] >> 4)
		key56o[3] = (key28l[3] << 4) | (key28r[0])
		key56o[4] = key28r[1]
		key56o[5] = key28r[2]
		key56o[6] = key28r[3]
		p.permutationChoose2(key56o, &(*outData)[i])
	}

}

func (p PowerDes) encry(inData []byte, subKey []byte, outData *[4]byte) {
	var i byte
	var outBuf = [6]byte{0, 0, 0, 0, 0, 0}
	var buf = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	p.expand(inData, &outBuf)
	for i = 0; i < 6; i++ {
		outBuf[i] = outBuf[i] ^ subKey[i]
	}
	buf[0] = outBuf[0] >> 2
	buf[1] = ((outBuf[0] & 0x03) << 4) | (outBuf[1] >> 4)
	buf[2] = ((outBuf[1] & 0x0f) << 2) | (outBuf[2] >> 6)
	buf[3] = outBuf[2] & 0x3f
	buf[4] = outBuf[3] >> 2
	buf[5] = ((outBuf[3] & 0x03) << 4) | (outBuf[4] >> 4)
	buf[6] = ((outBuf[4] & 0x0f) << 2) | (outBuf[5] >> 6)
	buf[7] = outBuf[5] & 0x3f
	for i = 0; i < 8; i++ {
		buf[i] = p.si(i, buf[i])
	}
	for i = 0; i < 4; i++ {
		outBuf[i] = (buf[i*2] << 4) | buf[i*2+1]
	}
	p.permutation(&outBuf)
	for i = 0; i < 4; i++ {
		(*outData)[i] = outBuf[i]
	}
}

func (p PowerDes) desData(desMode byte, inData []byte, outData *[]byte, subKey [][]byte) {
	var i, j int
	var temp = [4]byte{0, 0, 0, 0}
	var buf = [4]byte{0, 0, 0, 0}

	for i = 0; i < 8; i++ {
		(*outData)[i] = inData[i]
	}
	p.initPermutation(outData)

	if desMode == dmEncry {
		for i = 0; i < 16; i++ {
			for j = 0; j < 4; j++ {
				temp[j] = (*outData)[j]
			}
			for j = 0; j < 4; j++ {
				(*outData)[j] = (*outData)[j+4]
			}
			p.encry(*outData, subKey[i], &buf)
			for j = 0; j < 4; j++ {
				(*outData)[j+4] = temp[j] ^ buf[j]
			}
		}
		for j = 0; j < 4; j++ {
			temp[j] = (*outData)[j+4]
		}
		for j = 0; j < 4; j++ {
			(*outData)[j+4] = (*outData)[j]
		}
		for j = 0; j < 4; j++ {
			(*outData)[j] = temp[j]
		}
	} else {
		for i = 15; i >= 0; i-- {
			for j = 0; j < 4; j++ {
				temp[j] = (*outData)[j]
			}
			for j = 0; j < 4; j++ {
				(*outData)[j] = (*outData)[j+4]
			}
			p.encry(*outData, subKey[i], &buf)
			for j = 0; j < 4; j++ {
				(*outData)[j+4] = temp[j] ^ buf[j]
			}
		}
		for j = 0; j < 4; j++ {
			temp[j] = (*outData)[j+4]
		}
		for j = 0; j < 4; j++ {
			(*outData)[j+4] = (*outData)[j]
		}
		for j = 0; j < 4; j++ {
			(*outData)[j] = temp[j]
		}
	}
	p.conversePermutation(outData)
}

//EncryStr 加密
func (p PowerDes) EncryStr(aStr string, akey string) (encryStr string, err error) {

	if (len(aStr) > 0) && aStr[len(aStr)-1] == 0 {
		err = errors.New("最后一个字符为空")
		return "", err
	}

	var enc mahonia.Encoder
	enc = mahonia.NewEncoder("gbk")
	origDataStr := enc.ConvertString(aStr)
	strBuff := []byte(origDataStr)

	//strBuff := bytes.NewBufferString(aStr)
	keyBuff := bytes.NewBufferString(akey)

	if keyBuff.Len() < 8 {
		for {
			if keyBuff.Len() > 8 {
				break
			}
			keyBuff.WriteByte(0)
		}
	}
	for {
		// if strBuff.Len()%8 == 0 {
		// 	break
		// }
		// strBuff.WriteByte(0)
		if len(strBuff)%8 == 0 {
			break
		}
		strBuff = append(strBuff, 0)

	}

	//	aStr = strBuff.String()
	akey = keyBuff.String()

	var encryStrBuf = []byte{}
	var i, j int
	var subKey = [][]byte{}
	var strByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	var outByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	var keyByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}

	for i = 0; i < 16; i++ {
		var tmpArr []byte
		for j = 0; j < 6; j++ {
			tmpArr = append(tmpArr, 0)
		}
		subKey = append(subKey, tmpArr)
	}
	for j = 0; j < 8; j++ {
		keyByte[j] = akey[j]
	}
	p.makeKey(keyByte, &(subKey))

	for i = 0; i < len(strBuff)/8; i++ {
		for j = 0; j < 8; j++ {
			strByte[j] = strBuff[i*8+j]
		}
		p.desData(dmEncry, strByte, &outByte, subKey)
		for j = 0; j < 8; j++ {
			encryStrBuf = append(encryStrBuf, outByte[j])
		}
	}

	return string(encryStrBuf), nil
}

//DecryStr 解密
func (p PowerDes) DecryStr(aStr string, akey string) (decryStr string, err error) {

	keyBuff := bytes.NewBufferString(akey)

	if keyBuff.Len() < 8 {
		for {
			if keyBuff.Len() > 8 {
				break
			}
			keyBuff.WriteByte(0)
		}
	}

	akey = keyBuff.String()

	var decryStrBuf = []byte{}
	var i, j int
	var subKey = [][]byte{}
	var strByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	var outByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	var keyByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	for i = 0; i < 16; i++ {
		var tmpArr []byte
		for j = 0; j < 6; j++ {
			tmpArr = append(tmpArr, 0)
		}
		subKey = append(subKey, tmpArr)
	}

	for j = 0; j < 8; j++ {
		keyByte[j] = akey[j]
	}
	p.makeKey(keyByte, &(subKey))

	for i = 0; i < len(aStr)/8; i++ {
		for j = 0; j < 8; j++ {
			strByte[j] = aStr[i*8+j]
		}
		p.desData(dmDecry, strByte, &outByte, subKey)
		for j = 0; j < 8; j++ {
			decryStrBuf = append(decryStrBuf, outByte[j])
		}
	}

	for {

		if (len(decryStrBuf) > 0) && decryStrBuf[len(decryStrBuf)-1] == 0 {
			decryStrBuf = decryStrBuf[:len(decryStrBuf)-1]
		} else {
			break
		}
	}

	var dec mahonia.Decoder
	dec = mahonia.NewDecoder("gbk")
	decryStr = dec.ConvertString(string(decryStrBuf))

	return decryStr, nil
}

//PowerEncryStr 加密
func (p PowerDes) PowerEncryStr(aStr string, akey string) (encryStr string, err error) {

	encryStr, err = p.EncryStr(aStr, akey)
	if err != nil {
		return "", err
	}
	src := []byte(encryStr)
	encryStr = hex.EncodeToString(src)
	encryStr = strings.ToUpper(encryStr)
	return encryStr, nil
}

//PowerDecryStr 解密
func (p PowerDes) PowerDecryStr(aStr string, akey string) (decryStr string, err error) {

	var decryBuf []byte
	decryBuf, err = hex.DecodeString(aStr)
	if err != nil {
		return "", err
	}

	decryStr = string(decryBuf)
	decryStr, err = p.DecryStr(decryStr, akey)
	if err != nil {
		return "", err
	}
	return decryStr, nil
}
