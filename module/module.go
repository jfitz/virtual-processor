/*
Package module for virtual-processor
*/
package module

import (
	"errors"
	"github.com/jfitz/virtual-processor/vputils"
	"os"
)

// Module ------------------------
type Module struct {
	Properties       []vputils.NameValue
	CodePage         Page
	Exports          []vputils.NameValue
	DataPage         Page
	CodeAddressWidth int
	DataAddressWidth int
}

// Init - initialize
func (mod *Module) Init() {
}

// Write a module to a file
func (mod Module) Write(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", mod.Properties, f)
	vputils.WriteTextTable("exports", mod.Exports, f)
	vputils.WriteTextTable("code_properties", mod.CodePage.Properties, f)
	vputils.WriteBinaryBlock("code", mod.CodePage.Contents, f, mod.CodeAddressWidth)
	vputils.WriteTextTable("data_properties", mod.DataPage.Properties, f)
	vputils.WriteBinaryBlock("data", mod.DataPage.Contents, f, mod.DataAddressWidth)

	f.Sync()

	return nil
}

// Read a file into a module
func Read(moduleFile string) (Module, error) {
	f, err := os.Open(moduleFile)
	if err != nil {
		return Module{}, err
	}

	defer f.Close()

	header := vputils.ReadString(f)
	if header != "module" {
		return Module{}, errors.New("Did not find module header")
	}

	header = vputils.ReadString(f)
	if header != "properties" {
		return Module{}, errors.New("Did not find properties header")
	}

	properties, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	header = vputils.ReadString(f)
	if header != "exports" {
		return Module{}, errors.New("Did not find exports header")
	}

	exports, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	header = vputils.ReadString(f)
	if header != "code_properties" {
		return Module{}, errors.New("Did not find code_properties header")
	}

	codeProperties, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	codeAddressWidth := 1

	header = vputils.ReadString(f)
	if header != "code" {
		return Module{}, errors.New("Did not find code header")
	}

	code, err := vputils.ReadBinaryBlock(f, codeAddressWidth)
	if err != nil {
		return Module{}, err
	}

	codePage := Page{codeProperties, code, codeAddressWidth}

	header = vputils.ReadString(f)
	if header != "data_properties" {
		return Module{}, errors.New("Did not find data_properties header")
	}

	dataProperties, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	dataAddressWidth := 1

	header = vputils.ReadString(f)
	if header != "data" {
		return Module{}, errors.New("Did not find data header")
	}

	data, err := vputils.ReadBinaryBlock(f, dataAddressWidth)
	if err != nil {
		return Module{}, err
	}

	dataPage := Page{dataProperties, data, dataAddressWidth}

	// TODO: check data page datawidth is the same as code page datawidth

	mod := Module{
		Properties:       properties,
		CodePage:         codePage,
		Exports:          exports,
		DataPage:         dataPage,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}

	mod.Init()

	return mod, nil
}

// -------------------------------
