package zip

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestZip(t *testing.T) {
	Zip("/Users/cj/Desktop/DesktopGroups/GO项目/test/test.rar", "123", []string{"/Users/cj/Desktop/DesktopGroups/GO项目/test/test.xls", "/Users/cj/Desktop/DesktopGroups/GO项目/test/main.go"})
}

// password值可以为空""
func Zip(zipPath, password string, fileList []string) error {
	if len(fileList) < 1 {
		return fmt.Errorf("将要压缩的文件列表不能为空")
	}
	fz, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	zw := NewWriter(fz)
	defer zw.Close()
	for i, fileName := range fileList {
		err = func() error {
			fr, err := os.Open(fileName)
			if err != nil {
				return err
			}
			defer fr.Close()
			fileInfo, err := fr.Stat()
			if err != nil {
				return err
			}
			// 写入文件的头信息
			var w io.Writer
			if password != "" {
				var path string
				if i == 0 {
					path = "营业数据/" + fileInfo.Name()
				} else {
					path = "业绩数据/" + fileInfo.Name()
				}
				//压缩包加密
				w, err = zw.Encrypt(path, password, AES256Encryption)
			} else {
				w, err = zw.Create(fileInfo.Name())
			}

			if err != nil {
				return err
			}

			// 写入文件内容
			_, err = io.Copy(w, fr)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return zw.Flush()
}

func compress(file *os.File, prefix string, zw *Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := FileInfoHeader(info)
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
