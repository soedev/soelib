package des

import (
	"fmt"
	"testing"
)

func TestEntryptDesECB(t *testing.T) {
	p:=DecryptDESECB([]byte("BUPA1rbdsvue0O2VrrL/LNZl18AGvM6H"),[]byte("www.soe.xin"))
	fmt.Print(p)
}