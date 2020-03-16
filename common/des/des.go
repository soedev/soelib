package des

/*
Java默认DES算法使用DES/ECB/PKCS5Padding，而golang认为这种方式是不安全的，
所以故意没有提供这种加密方式，那如果我们还是要用到怎么办？
下面贴上golang版的DES ECB加密解密代码（默认对密文做了base64处理）。
*/

import (
	"bytes"
	"crypto/des"
	"encoding/base64"
	"fmt"
	"log"
)

var DesKey = []byte("www.soe.xin")

//加密
func EntryptDesECB(data, key []byte) string {
	if len(key) > 8 {
		key = key[:8]
	}
	block, err := des.NewCipher(key)
	if err != nil {
		log.Printf("EntryptDesECB newCipher error[%v]", err)
		return ""
	}
	bs := block.BlockSize()
	data = PKCS5Padding(data, bs)
	if len(data)%bs != 0 {
		log.Println("EntryptDesECB Need a multiple of the blocksize")
		return ""
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		block.Encrypt(dst, data[:bs])
		data = data[bs:]
		dst = dst[bs:]
	}
	return base64.StdEncoding.EncodeToString(out)
}

//解密
func DecryptDESECB(d, key []byte) string {
	//source := string(d[:])
	data, err := base64.StdEncoding.DecodeString(string(d[:]))
	if err != nil {
		fmt.Println("DecryptDES Decode base64 error")
		return ""
	}
	if len(key) > 8 {
		key = key[:8]
	}
	block, err := des.NewCipher(key)
	if err != nil {
		fmt.Println("DecryptDES NewCipher error")
		return ""
	}
	bs := block.BlockSize()
	if len(data)%bs != 0 {
		fmt.Println("DecryptDES crypto/cipher: input not full blocks")
		return ""
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		block.Decrypt(dst, data[:bs])
		data = data[bs:]
		dst = dst[bs:]
	}
	out = PKCS5UnPadding(out)
	return string(out)
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
