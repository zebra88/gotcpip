package util

import (
	"reflect"
	"regexp"
	"strconv"
	"unsafe"
)

func VerifyChecksum(data []byte) bool {
	return Checksum(data) == 0
}

func Byte2uint16(b []byte) uint16 {
	return uint16((uint16(b[1]) << 8) | (uint16(b[0])))
}

func Byte2int16(b []byte) int16 {
	return int16((uint16(b[1]) << 8) | (uint16(b[0])))
}

func Byte2uint32(b []byte) uint32 {
	return uint32((uint32(b[3]) << 24) | (uint32(b[2]) << 16) | (uint32(b[1]) << 8) | uint32(b[0]))
}

func Byte2int32(b []byte) int32 {
	return int32((uint32(b[3]) << 24) | (uint32(b[2]) << 16) | (uint32(b[1]) << 8) | uint32(b[0]))
}

func Byte2Pointer(b []byte) unsafe.Pointer {
	return unsafe.Pointer(
		(*reflect.SliceHeader)(unsafe.Pointer(&b)).Data,
	)
	//	ptr = (*IPPacket)(unsafe.Pointer(((*reflect.SliceHeader)(unsafe.Pointer(&b)).Data)))
	//	fmt.Println("%p", ptr)
	//	return ptr
}
func Ip2long(ipstr string) (ip uint32) {
	r := `^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})`
	reg, err := regexp.Compile(r)
	if err != nil {
		return
	}
	ips := reg.FindStringSubmatch(ipstr)
	if ips == nil {
		return
	}
	/*
		ip1, _ := strconv.Atoi(ips[1])
		ip2, _ := strconv.Atoi(ips[2])
		ip3, _ := strconv.Atoi(ips[3])
		ip4, _ := strconv.Atoi(ips[4])
	*/
	ip4, _ := strconv.Atoi(ips[1])
	ip3, _ := strconv.Atoi(ips[2])
	ip2, _ := strconv.Atoi(ips[3])
	ip1, _ := strconv.Atoi(ips[4])

	if ip1 > 255 || ip2 > 255 || ip3 > 255 || ip4 > 255 {
		return
	}

	ip += uint32(ip1 * 0x1000000)
	ip += uint32(ip2 * 0x10000)
	ip += uint32(ip3 * 0x100)
	ip += uint32(ip4)

	return
}
func Ntohs(a uint16) uint16 {
	return (a&0xff)<<8 | (a >> 8)
}

func Ntohl(a uint32) uint32 {
	return (a&0xff)<<24 | (((a & 0xff00) << 8) & 0xff0000) | (((a & 0xff0000) >> 8) & 0xff00) | (a >> 24)
}
func Htons(a uint16) uint16 {
	return (a&0xff)<<8 | (a >> 8)
}

func Htonl(a uint32) uint32 {
	return (a&0xff)<<24 | (((a & 0xff00) << 8) & 0xff0000) | (((a & 0xff0000) >> 8) & 0xff00) | (a >> 24)
}

/*
var sizeOfMyStruct = int(unsafe.Sizeof(MyStruct{}))

func Pointer2Bytes(pointer unsafe.Pointer) []byte {
	var x reflect.SliceHeader
	x.Len = sizeOfMyStruct
	x.Cap = sizeOfMyStruct
	x.Data = uintptr(unsafe.Pointer(s))
	return *(*[]byte)(unsafe.Pointer(&x))
}
*/
// Calculate 16-bit 1s complement additive checksum
func Checksum(data []byte) uint16 {
	var chksum uint32

	var lsb uint16
	var msb uint16

	// 32-bit sum (2's complement sum of 16 bits with carry)
	for i := 0; i < len(data)-1; i += 2 {
		msb = uint16(data[i])
		lsb = uint16(data[i+1])
		chksum += uint32(lsb + (msb << 8))
	}

	// 1's complement 16-bit sum via "end arround carry" of 2's complement
	chksum = ((chksum >> 16) & 0xFFFF) + (chksum & 0xFFFF)

	return uint16(0xFFFF & (^chksum))
}
