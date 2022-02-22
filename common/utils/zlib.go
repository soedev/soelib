package utils

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
)

//DoZlibCompress 进行zlib压缩
func DoZlibCompress(src []byte) string {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	//使用base64编码
	encoded := base64.StdEncoding.EncodeToString(in.Bytes())
	return encoded
}
//DoZlibUnCompress 进行zlib解压缩
func DoZlibUnCompress(compressSrc []byte) string {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)
	return string(out.Bytes())
}