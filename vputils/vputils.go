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
	return IsAlnum(c) || c == '.' || c == ':'
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
		// quote by itself accepts anything
		return true
	}

	if IsDoubleQuote(token[0]) && len(token) > 1 && !IsDoubleQuote(token[len(token)-1]) {
		// quote with non-quote characters accepts anything
		return true
	}

	if IsDigit(token[0]) {
		// numeric token accepts digits
		return IsDigit(c)
	}

	if IsAlpha(token[0]) {
		// text token accepts alpha and digit and underscore and colon
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

func ReadFile(sourceFile string) []string {
	b, err := ioutil.ReadFile(sourceFile)
	CheckAndPanic(err)

	source := string(b)
	sourceLines := strings.Split(source, "\n")

	return sourceLines
}

// --------------------
// address
// --------------------
type Address struct {
	Bytes   []byte
	Maximum int
}

// --------------------
func MakeAddress(value int, size int, maximum int) (Address, error) {
	if value < 0 {
		return Address{[]byte{}, 0}, errors.New("Negative address")
	}

	if value > maximum {
		return Address{[]byte{}, 0}, errors.New("Address exceeds maximum")
	}

	addressBytes := []byte{}

	for i := 0; i < size; i++ {
		b := byte(value & 0xff)
		addressBytes = append(addressBytes, b)
		value = value / 256
	}

	return Address{addressBytes, maximum}, nil
}

// --------------------
func (address Address) Empty() bool {
	return len(address.Bytes) == 0
}

// --------------------
func (address Address) NumBytes() int {
	return len(address.Bytes)
}

// --------------------
func (address Address) ToInt() int {
	value := 0
	for _, b := range address.Bytes {
		// should shift here
		// little-endian or big-endian?
		value += int(b)
	}

	return value
}

// --------------------
func (address Address) ToString() string {
	s := ""
	for _, b := range address.Bytes {
		s += fmt.Sprintf("%02X", b)
	}

	return s
}

// --------------------
func (address Address) ByteValue() byte {
	return address.Bytes[0]
}

// --------------------
func (address Address) AddByte(i int) Address {
	increment := byte(i)
	a := address.ByteValue() + increment
	as := []byte{a}
	return Address{as, address.Maximum}
}

// --------------------
// vector
// --------------------
type Vector []byte

// --------------------
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

// --------------------
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

// --------------------
// --------------------

// --------------------
// bool stack
// --------------------
type BoolStack []bool

// --------------------
func (stack BoolStack) Push(v bool) BoolStack {
	return append(stack, v)
}

// --------------------
func (stack BoolStack) Top() (bool, error) {
	if len(stack) < 1 {
		return false, errors.New("Stack underflow")
	}

	last := len(stack) - 1
	return stack[last], nil
}

// --------------------
func (stack BoolStack) Pop() (bool, BoolStack, error) {
	if len(stack) < 1 {
		return false, stack, errors.New("Stack underflow")
	}

	last := len(stack) - 1
	return stack[last], stack[:last], nil
}

// --------------------
// --------------------

// --------------------
// byte stack
// --------------------
type ByteStack []byte

// --------------------
func (stack ByteStack) PushByte(v byte) ByteStack {
	return append(stack, v)
}

// --------------------
func reverseBytes(bs []byte) []byte {
	last := len(bs) - 1

	for i := 0; i < len(bs)/2; i++ {
		bs[i], bs[last-i] = bs[last-i], bs[i]
	}

	return bs
}

// --------------------
func (stack ByteStack) PushBytes(vs []byte) ByteStack {
	bs := reverseBytes(vs)
	return append(stack, bs...)
}

// --------------------
func (stack ByteStack) TopByte() (byte, error) {
	count := 1
	if len(stack) < count {
		return 0, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], nil
}

// --------------------
func (stack ByteStack) PopByte(count int) ([]byte, ByteStack, error) {
	if len(stack) < count {
		return []byte{}, stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last:], stack[:last], nil
}

// --------------------
func (stack ByteStack) PushString(s string) ByteStack {
	bs := []byte(s)
	stack = stack.PushBytes(bs)
	b := byte(len(s))
	stack = stack.PushByte(b)

	return stack
}

// --------------------
func (stack ByteStack) PopString() (string, ByteStack) {
	// pop size of name
	counts, stack, err := stack.PopByte(1)
	CheckAndExit(err)
	count := int(counts[0])

	// pop bytes that make the string
	bytes := []byte{}
	s := ""
	for i := 0; i < count; i++ {
		bytes, stack, err = stack.PopByte(1)
		CheckAndExit(err)
		if bytes[0] != 0 {
			s += string(bytes[0])
		}
	}

	return s, stack
}

// --------------------
// --------------------

// --------------------
// address stack
// --------------------
type AddressStack []Address

// --------------------
func (stack AddressStack) Push(address Address) AddressStack {
	return append(stack, address)
}

// --------------------
func (stack AddressStack) Top() (Address, error) {
	count := 1
	if len(stack) < count {
		return Address{[]byte{}, 0}, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], nil
}

// --------------------
func (stack AddressStack) Pop() (AddressStack, error) {
	count := 1
	if len(stack) < count {
		return stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[:last], nil
}

// --------------------
func (stack AddressStack) TopPop() (Address, AddressStack, error) {
	count := 1
	if len(stack) < count {
		return Address{[]byte{}, 0}, stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], stack[:last], nil
}
