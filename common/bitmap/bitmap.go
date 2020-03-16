package bitmap

import (
	"fmt"
)

// 暂时只支持 1 << 32 位（可以扩展到 1 << 64)
// The Max Size is 0x01 << 32 at present(can expand to 0x01 << 64)
const BitmapSize = 0x01 << 32

// Bitmap 数据结构定义
type Bitmap struct {
	// 保存实际的 bit 数据
	Data []byte `json:"data"`
	// 指示该 Bitmap 的 bit 容量
	Bitsize uint64 `json:"bitsize"`
	// 该 Bitmap 被设置为 1 的最大位置（方便遍历）
	Maxpos uint64 `json:"maxpos"`
}

// NewBitmap 使用默认容量实例化一个 Bitmap
func NewBitmap() *Bitmap {
	return NewBitmapSize(BitmapSize)
}

// NewBitmapSize 根据指定的 size 实例化一个 Bitmap
func NewBitmapSize(size int) *Bitmap {
	if size == 0 || size > BitmapSize {
		size = BitmapSize
	} else if remainder := size % 8; remainder != 0 {
		size += 8 - remainder
	}

	return &Bitmap{Data: make([]byte, size>>3), Bitsize: uint64(size - 1)}
}

// SetBit 将 offset 位置的 bit 置为 value (0/1)
func (this *Bitmap) SetBit(offset uint64, value uint8) bool {
	index, pos := offset/8, offset%8

	if this.Bitsize < offset {
		return false
	}

	if value == 0 {
		// &^ 清位
		this.Data[index] &^= 0x01 << pos
	} else {
		this.Data[index] |= 0x01 << pos

		// 记录曾经设置为 1 的最大位置
		if this.Maxpos < offset {
			this.Maxpos = offset
		}
	}

	return true
}

// GetBit 获得 offset 位置处的 value
func (this *Bitmap) GetBit(offset uint64) uint8 {
	index, pos := offset/8, offset%8

	if this.Bitsize < offset {
		return 0
	}

	return (this.Data[index] >> pos) & 0x01
}

// Maxpos 获的置为 1 的最大位置
func (this *Bitmap) GetMaxpos() uint64 {
	return this.Maxpos
}

//获取 为1的 个数
func (this *Bitmap) Count() int {
	num := 0
	for _, byte := range this.Data {
		num += costCount(int(byte))
	}
	return num
}

// String 实现 Stringer 接口（只输出开始的100个元素）
func (this *Bitmap) String() string {
	var maxTotal, bitTotal uint64 = 100, this.Maxpos + 1

	if this.Maxpos > maxTotal {
		bitTotal = maxTotal
	}

	numSlice := make([]uint64, 0, bitTotal)

	var offset uint64
	for offset = 0; offset < bitTotal; offset++ {
		if this.GetBit(offset) == 1 {
			numSlice = append(numSlice, offset)
		}
	}

	return fmt.Sprintf("%v", numSlice)
}

//统计二进制中 位数 为1 的个数
func costCount(v int) int {
	if v <= 0 {
		return 0
	}
	num := 0
	for {
		v &= (v - 1)
		num++
		if v <= 0 {
			break
		}
	}
	return num
}
