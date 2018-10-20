/*
Package module for virtual-processor
*/
package module

import (
	"errors"
	"github.com/jfitz/virtual-processor/vputils"
)

// Page --------------------------
type Page struct {
	Properties   []vputils.NameValue
	Contents     vputils.Vector
	AddressWidth int
}

// GetAddress - get bytes and convert to an address
func (page Page) GetAddress(address vputils.Address, addressWidth int, maximum int) (vputils.Address, error) {
	emptyAddress, _ := vputils.MakeAddress(0, addressWidth, 0)

	addrBytes, err := page.Contents.GetBytes(address, addressWidth)
	if err != nil {
		return emptyAddress, err
	}

	resultAddress, err := vputils.BytesToAddress(addrBytes, maximum)
	if err != nil {
		message := "Cannot get address: " + err.Error()
		return emptyAddress, errors.New(message)
	}

	return resultAddress, nil
}

// GetOpcode - get the opcode at PC
func (code Page) GetOpcode(pc vputils.Address) (byte, error) {
	return code.Contents.GetByte(pc)
}

// ImmediateByte - get a byte
func (code Page) ImmediateByte(pc vputils.Address) ([]byte, error) {
	codeAddress := pc.Increment(1)

	value, err := code.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	return []byte{value}, nil
}

// ImmediateInt - get an I16
func (code Page) ImmediateInt(pc vputils.Address) ([]byte, error) {
	codeAddress := pc.Increment(1)

	values := []byte{}

	value, err := code.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	values = append(values, value)

	codeAddress = codeAddress.Increment(1)

	value, err = code.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	values = append(values, value)

	return values, nil
}

// JumpAddress - get direct address
func (code Page) JumpAddress(pc vputils.Address) (vputils.Address, error) {
	codeAddress := pc.Increment(1)

	jumpAddress, err := code.GetAddress(codeAddress, code.AddressWidth, len(code.Contents))

	return jumpAddress, err
}

// DirectAddress - get direct address
func (code Page) DirectAddress(pc vputils.Address, data Page) (vputils.Address, error) {
	codeAddress := pc.Increment(1)

	dataAddress, err := code.GetAddress(codeAddress, data.AddressWidth, len(data.Contents))

	return dataAddress, err
}

// DirectByte - get byte via direct address
func (code Page) DirectByte(pc vputils.Address, data Page) (byte, error) {
	dataAddress, err := code.DirectAddress(pc, data)
	if err != nil {
		return 0, err
	}

	value, err := data.Contents.GetByte(dataAddress)
	if err != nil {
		return 0, err
	}

	return value, nil
}

// IndirectAddress - get indirect address
func (code Page) IndirectAddress(pc vputils.Address, data Page) (vputils.Address, error) {
	codeAddress := pc.Increment(1)

	addressWidth := data.AddressWidth
	dataAddress1, err := code.GetAddress(codeAddress, addressWidth, len(data.Contents))
	if err != nil {
		return dataAddress1, err
	}

	dataAddress, err := data.GetAddress(dataAddress1, addressWidth, len(data.Contents))

	return dataAddress, err
}

// IndirectByte - get byte via indirect address
func (code Page) IndirectByte(pc vputils.Address, data Page) (byte, error) {
	dataAddress, err := code.IndirectAddress(pc, data)
	if err != nil {
		return 0, err
	}

	value, err := data.Contents.GetByte(dataAddress)
	if err != nil {
		return 0, err
	}

	return value, nil
}

// GetConditionals - get the conditionals for instruction at PC
func (code Page) GetConditionals(pc vputils.Address) (Conditionals, error) {
	conditionals := Conditionals{}
	err := errors.New("")

	myByte, err := code.Contents.GetByte(pc)
	if err != nil {
		return conditionals, err
	}

	hasConditional := true

	for hasConditional {
		if myByte >= 0xE0 && myByte <= 0xEF {
			conditionals = append(conditionals, myByte)
			// step PC over conditionals
			pc = pc.Increment(1)
			myByte, err = code.Contents.GetByte(pc)
			if err != nil {
				return conditionals, err
			}
		} else {
			hasConditional = false
		}
	}

	return conditionals, nil
}
