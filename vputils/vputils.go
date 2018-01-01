/*
package of utilities for virtual-processor
*/
package vputils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type NameValue struct {
	Name  string
	Value string
}

func CheckAndPanic(e error) {
	if e != nil {
		panic(e)
	}
}

func CheckAndExit(e error) {
	if e != nil {
		fmt.Println(e.Error())
		os.Exit(1)
	}
}

func CheckPrintAndExit(e error, message string) {
	if e != nil {
		fmt.Println(e.Error() + " " + message)
		os.Exit(1)
	}
}

func checkWidth(width int) {
	if width != 1 && width != 2 {
		CheckAndExit(errors.New("Invalid width"))
	}
}

func IsSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

func IsDoubleQuote(c byte) bool {
	return c == '"'
}

func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func IsAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func IsAlnum(c byte) bool {
	return IsDigit(c) || IsAlpha(c) || c == '_'
}

func IsUpper(c byte) bool {
	return (c >= 'A' && c <= 'Z')
}

func IsLower(c byte) bool {
	return (c >= 'a' && c <= 'z')
}

func IsText(c byte) bool {
	return IsAlnum(c) || c == '.'
}

func IsDirectAddress(s string) bool {
	return len(s) >= 2 && s[0] == '@' && IsAlpha(s[1])
}

func IsIndirectAddress(s string) bool {
	return len(s) >= 3 && s[0] == '@' && s[1] == '@' && IsAlpha(s[2])
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

	if IsDoubleQuote(token[0]) && len(token) == 1 {
		return true
	}

	if IsDoubleQuote(token[0]) && len(token) > 1 && !IsDoubleQuote(token[len(token)-1]) {
		return true
	}

	if IsDigit(token[0]) {
		// numeric token accepts digits
		return IsDigit(c)
	}

	if IsAlpha(token[0]) {
		// text token accepts alpha and digit
		return IsText(c)
	}

	if token == "@" {
		return c == '@' || IsAlpha(c)
	}

	if token == "@@" {
		return IsAlpha(c)
	}

	// after checking for '@' and '@@', accept text (for labels)
	// we know there is at least one leading alpha
	if token[0] == '@' {
		return IsAlnum(c)
	}

	return false
}

func Tokenize(s string) []string {
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
	CheckAndPanic(err)

	value := int(bytes[0])

	return value
}

func read2ByteInt(f *os.File) int {
	bytes := make([]byte, 2)
	_, err := f.Read(bytes)
	CheckAndPanic(err)

	value := int(bytes[1])<<8 + int(bytes[0])

	return value
}

// if value is greater than 255 then error
func write1ByteInt(f *os.File, value int) {
	low := byte(value & 0x00ff)
	bytes := []byte{low}

	_, err := f.Write(bytes)
	CheckAndPanic(err)
}

// if value is greater than 65535 then error
func write2ByteInt(f *os.File, value int) {
	high := byte(value & 0xff00 >> 8)
	low := byte(value & 0x00ff)
	bytes := []byte{low, high}

	_, err := f.Write(bytes)
	CheckAndPanic(err)
}

func ReadString(f *os.File) string {
	bytes := []byte{}
	one_byte := make([]byte, 1)
	one_byte[0] = 1
	for one_byte[0] != 0 {
		_, err := f.Read(one_byte)
		CheckAndPanic(err)
		if one_byte[0] != 0 {
			bytes = append(bytes, one_byte...)
		}
	}

	name := string(bytes)

	return name
}

func WriteString(f *os.File, text string) {
	_, err := f.Write([]byte(text))
	CheckAndPanic(err)

	zero_byte := []byte{0}

	_, err = f.Write(zero_byte)
	CheckAndPanic(err)
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
	CheckAndPanic(err)

	checkCountBytes := 0
	switch width {
	case 1:
		checkCountBytes = read1ByteInt(f)
	case 2:
		checkCountBytes = read2ByteInt(f)
	}

	if checkCountBytes != countBytes {
		CheckAndExit(errors.New("Block count error"))
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
	CheckAndPanic(err)

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
	CheckAndPanic(err)

	if one_byte[0] != stx_byte[0] {
		CheckAndExit(errors.New("Did not find STX"))
	}

	// read until ETX
	bytes := []byte{}
	one_byte[0] = 0
	for one_byte[0] != etx_byte[0] {
		_, err := f.Read(one_byte)
		CheckAndPanic(err)
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
	CheckAndPanic(err)

	for _, nameValue := range table {
		name := []byte(nameValue.Name)
		value := []byte(nameValue.Value)

		// write name
		_, err = f.Write(name)
		CheckAndPanic(err)
		// write FS
		_, err = f.Write(fs_byte)
		CheckAndPanic(err)
		// write value
		_, err = f.Write(value)
		CheckAndPanic(err)
		// write RS (0x1e)
		_, err = f.Write(rs_byte)
		CheckAndPanic(err)
	}
	// write ETX
	_, err = f.Write(etx_byte)
	CheckAndPanic(err)
}

type Address struct {
	Bytes []byte
}

func MakeAddress(value int, size int) Address {
	address := []byte{}

	for i := 0; i < size; i++ {
		b := byte(value & 0xff)
		address = append(address, b)
		value = value / 256
	}

	return Address{address}
}

func (address Address) Empty() bool {
	return len(address.Bytes) == 0
}

func (address Address) Size() int {
	return len(address.Bytes)
}

func (address Address) ToInt() int {
	value := 0
	for _, b := range address.Bytes {
		// should shift here
		// little-endian or big-endian?
		value += int(b)
	}

	return value
}

func (address Address) ToString() string {
	s := ""
	for _, b := range address.Bytes {
		s += fmt.Sprintf("%02X", b)
	}

	return s
}

func (address Address) ByteValue() byte {
	return address.Bytes[0]
}

func (ca Address) AddByte(i int) Address {
	b := byte(i)
	a := ca.ByteValue() + b
	as := []byte{a}
	return Address{as}
}

type Vector []byte

func (v Vector) GetByte(address Address) (byte, error) {
	max := len(v) - 1
	offset := address.ToInt()
	if offset < 0 || offset > max {
		off := strconv.Itoa(offset)
		maxs := strconv.Itoa(max)
		return 0, errors.New("Index " + off + " out of range [0.." + maxs + "]")
	}

	value := v[offset]
	return value, nil
}

func (v Vector) PutByte(address Address, value byte) error {
	max := len(v) - 1
	offset := address.ToInt()
	if offset < 0 || offset > max {
		off := strconv.Itoa(offset)
		maxs := strconv.Itoa(max)
		return errors.New("Index " + off + " out of range [0.." + maxs + "]")
	}

	v[offset] = value

	return nil
}

type addressStack []Address

func (s addressStack) push(address Address) addressStack {
	return append(s, address)
}

func (s addressStack) top() (Address, error) {
	if len(s) == 0 {
		return Address{[]byte{}}, errors.New("Stack underflow")
	}

	last := len(s) - 1
	return s[last], nil
}

func (s addressStack) pop() (addressStack, error) {
	if len(s) == 0 {
		return s, errors.New("Stack underflow")
	}

	last := len(s) - 1
	return s[:last], nil
}

func (s addressStack) toppop() (Address, addressStack, error) {
	if len(s) == 0 {
		return Address{[]byte{}}, s, errors.New("Stack underflow")
	}

	last := len(s) - 1
	return s[last], s[:last], nil
}

type Module struct {
	Properties       []NameValue
	Code             Vector
	Exports          []NameValue
	Data             Vector
	CodeAddressWidth int
	DataAddressWidth int
	pc               Address
	RetStack         addressStack
}

func (module *Module) Init() {
}

func (module *Module) SetPC(address Address) {
	module.pc = address
}

func (module Module) PCByteValue() byte {
	return module.pc.ByteValue()
}

func (module Module) PC() Address {
	return module.pc
}

func (module Module) ImmediateByte() []byte {
	codeAddress := module.pc.AddByte(1)

	value, err := module.Code.GetByte(codeAddress)
	CheckAndExit(err)

	return []byte{value}
}

func (module Module) ImmediateInt() []byte {
	codeAddress := module.pc.AddByte(1)

	values := []byte{}

	value, err := module.Code.GetByte(codeAddress)
	CheckAndExit(err)
	values = append(values, value)

	codeAddress = codeAddress.AddByte(1)

	value, err = module.Code.GetByte(codeAddress)
	CheckAndExit(err)
	values = append(values, value)

	return values
}

func (module Module) DirectAddress() Address {
	codeAddress := module.pc.AddByte(1)

	dataAddr, err := module.Code.GetByte(codeAddress)
	CheckAndExit(err)
	da := []byte{dataAddr}
	dataAddress := Address{da}

	return dataAddress
}

func (module Module) OffsetAddress() Address {
	codeAddress := module.pc.AddByte(1)

	offset, err := module.Code.GetByte(codeAddress)
	CheckAndExit(err)
	dataAddress := module.pc.AddByte(int(offset))

	return dataAddress
}

func (module Module) DirectByte() (byte, Address) {
	dataAddress := module.DirectAddress()

	value, err := module.Data.GetByte(dataAddress)
	CheckAndExit(err)

	return value, dataAddress
}

func (module Module) IndirectAddress() Address {
	dataAddress := module.DirectAddress()
	dataAddr, err := module.Data.GetByte(dataAddress)
	CheckAndExit(err)
	da := []byte{dataAddr}
	dataAddress = Address{da}

	return dataAddress
}

func (module Module) IndirectByte() (byte, Address) {
	dataAddress := module.IndirectAddress()
	value, err := module.Data.GetByte(dataAddress)
	CheckAndExit(err)

	return value, dataAddress
}

func (module *Module) Push(address Address) {
	module.RetStack = module.RetStack.push(address)
}

func (module *Module) TopPop() (Address, error) {
	address, retStack, err := module.RetStack.toppop()
	module.RetStack = retStack

	return address, err
}

func ReadFile(sourceFile string) []string {
	b, err := ioutil.ReadFile(sourceFile)
	CheckAndPanic(err)

	source := string(b)
	sourceLines := strings.Split(source, "\n")

	return sourceLines
}
