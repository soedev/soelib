package utils

/**
  color   颜色处理工具类
*/

import "strconv"

type RGB struct {
	R, G, B int64
}

//ColorHex  颜色16进制值
type ColorHex struct {
	Str string
}

func t2x(t int64) string {
	result := strconv.FormatInt(t, 16)
	if len(result) == 1 {
		result = "0" + result
	}
	return result
}

func (color RGB) RGB2HEX() ColorHex {
	r := t2x(color.R)
	g := t2x(color.G)
	b := t2x(color.B)
	return ColorHex{r + g + b}
}

func (color ColorHex) HEX2RGB() RGB {
	r, _ := strconv.ParseInt(color.Str[:2], 16, 32)
	g, _ := strconv.ParseInt(color.Str[2:4], 16, 32)
	b, _ := strconv.ParseInt(color.Str[4:], 16, 32)
	return RGB{r, g, b}
}

//颜色转成 565 数值
func (color RGB) RGB2RGB565() int64 {
	r := (color.R >> 3) & 0x1F
	g := (color.G >> 2) & 0x3F
	b := (color.B >> 3) & 0x1F
	d := b + (g << 5) + (r << 11)
	return d
}

func RGB5652RGB(c int64) RGB {
	r := (c & 0xF800) >> 8
	g := (c & 0x07E0) >> 3
	b := (c & 0x001F) << 3
	return RGB{r, g, b}
}
