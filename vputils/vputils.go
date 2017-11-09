/*
package of utilities for virtual-processor
*/
package vputils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type NameValue struct {
	Name  string
	Value string
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func ShowErrorAndStop(message string) {
	if message != "" {
		fmt.Println(message)
		os.Exit(1)
	}
}

func checkWidth(width int) {
	if width != 1 && width != 2 {
		ShowErrorAndStop("Invalid width")
	}
}

func IsSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func IsAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func IsAlnum(c byte) bool {
	return IsDigit(c) || IsAlpha(c)
}

// test for compatible character
func compatible(token string, c byte) bool {
	if token == "" {
		// empty token accepts anything
		return true
	}

	if IsSpace(token[0]) {
		// space token accepts spaces
		return IsSpace(c)
	}

	if IsDigit(token[0]) {
		// numeric token accepts digits
		return IsDigit(c)
	}

	if IsAlpha(token[0]) {
		// text token accepts alpha and digit
		return IsAlnum(c)
	}

	return false
}

func Split(s string) []string {
	parts := []string{}
	current := ""
	for _, c := range s {
		if compatible(current, byte(c)) {
			current += string(c)
		} else {
			// incompatible character requires a new token
			parts = append(parts, current)
			current = string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func read1ByteInt(f *os.File) int {
	bytes := make([]byte, 1)
	_, err := f.Read(bytes)
	Check(err)

	value := int(bytes[0])

	return value
}

func read2ByteInt(f *os.File) int {
	bytes := make([]byte, 2)
	_, err := f.Read(bytes)
	Check(err)

	value := int(bytes[1])<<8 + int(bytes[0])

	return value
}

// if value is greater than 255 then error
func write1ByteInt(f *os.File, value int) {
	low := byte(value & 0x00ff)
	bytes := []byte{low}

	_, err := f.Write(bytes)
	Check(err)
}

// if value is greater than 65535 then error
func write2ByteInt(f *os.File, value int) {
	high := byte(value & 0xff00 >> 8)
	low := byte(value & 0x00ff)
	bytes := []byte{low, high}

	_, err := f.Write(bytes)
	Check(err)
}

func ReadString(f *os.File) string {
	bytes := []byte{}
	one_byte := make([]byte, 1)
	one_byte[0] = 1
	for one_byte[0] != 0 {
		_, err := f.Read(one_byte)
		Check(err)
		if one_byte[0] != 0 {
			bytes = append(bytes, one_byte...)
		}
	}

	name := string(bytes)

	return name
}

func WriteString(f *os.File, text string) {
	_, err := f.Write([]byte(text))
	Check(err)

	zero_byte := []byte{0}

	_, err = f.Write(zero_byte)
	Check(err)
}

func ReadBinaryBlock(f *os.File, width int) []byte {
	checkWidth(width)

	countBytes := 0
	switch width {
	case 1:
		countBytes = read1ByteInt(f)
	case 2:
		countBytes = read2ByteInt(f)
	}

	code := make([]byte, countBytes)
	_, err := f.Read(code)
	Check(err)

	checkCountBytes := 0
	switch width {
	case 1:
		checkCountBytes = read1ByteInt(f)
	case 2:
		checkCountBytes = read2ByteInt(f)
	}

	if checkCountBytes != countBytes {
		ShowErrorAndStop("Block count error")
	}

	return code
}

func WriteBinaryBlock(name string, bytes []byte, f *os.File, width int) {
	checkWidth(width)

	WriteString(f, name)
	switch width {
	case 1:
		write1ByteInt(f, len(bytes))
	case 2:
		write2ByteInt(f, len(bytes))
	}

	_, err := f.Write(bytes)
	Check(err)

	switch width {
	case 1:
		write1ByteInt(f, len(bytes))
	case 2:
		write2ByteInt(f, len(bytes))
	}
}

func ReadTextTable(f *os.File) []NameValue {
	stx_byte := []byte{0x02}
	etx_byte := []byte{0x03}
	fs_byte := []byte{0x1c}
	rs_byte := []byte{0x1e}

	one_byte := make([]byte, 1)

	// read STX
	_, err := f.Read(one_byte)
	Check(err)

	if one_byte[0] != stx_byte[0] {
		ShowErrorAndStop("Did not find STX")
	}

	// read until ETX
	bytes := []byte{}
	one_byte[0] = 0
	for one_byte[0] != etx_byte[0] {
		_, err := f.Read(one_byte)
		Check(err)
		if one_byte[0] != etx_byte[0] {
			bytes = append(bytes, one_byte...)
		}
	}

	all_text := string(bytes)
	records := strings.Split(all_text, string(rs_byte))

	nameValues := []NameValue{}

	for _, record := range records {
		fields := strings.Split(record, string(fs_byte))
		if len(fields) == 2 {
			name := fields[0]
			value := fields[1]
			nameValue := NameValue{name, value}
			nameValues = append(nameValues, nameValue)
		}
	}

	return nameValues
}

func WriteTextTable(name string, table []NameValue, f *os.File) {
	WriteString(f, name)

	stx_byte := []byte{0x02}
	etx_byte := []byte{0x03}
	fs_byte := []byte{0x1c}
	rs_byte := []byte{0x1e}

	// write STX
	_, err := f.Write(stx_byte)
	Check(err)

	for _, nameValue := range table {
		name := []byte(nameValue.Name)
		value := []byte(nameValue.Value)

		// write name
		_, err = f.Write(name)
		Check(err)
		// write FS
		_, err = f.Write(fs_byte)
		Check(err)
		// write value
		_, err = f.Write(value)
		Check(err)
		// write RS (0x1e)
		_, err = f.Write(rs_byte)
		Check(err)
	}
	// write ETX
	_, err = f.Write(etx_byte)
	Check(err)
}

func ReadFile(sourceFile string) []string {
	fmt.Printf("Reading file %s...\n", sourceFile)
	b, err := ioutil.ReadFile(sourceFile)
	Check(err)

	source := string(b)
	sourceLines := strings.Split(source, "\n")

	return sourceLines
}
