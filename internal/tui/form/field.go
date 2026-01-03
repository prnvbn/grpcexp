package form

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type FieldKind int

const (
	FieldText FieldKind = iota
	FieldBool
	FieldEnum
)

type Field struct {
	name string
	Kind FieldKind

	textInput textinput.Model // for text fields
	enumList  list.Model      // for enum and bool fields

	validate func(string) error
}

type EnumItem struct {
	Name  string
	Value string
}

func (i EnumItem) Title() string       { return i.Name }
func (i EnumItem) Description() string { return "" }
func (i EnumItem) FilterValue() string { return i.Name }

func NewTextField(name, placeholder string, charLimit int, validate func(string) error) *Field {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	return &Field{
		name:      name,
		Kind:      FieldText,
		textInput: ti,
		validate:  validate,
	}
}

func NewBoolField(name string) *Field {
	items := []list.Item{
		EnumItem{Name: "false", Value: "false"},
		EnumItem{Name: "true", Value: "true"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(items, delegate, 30, 5)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)

	return &Field{
		name:     name,
		Kind:     FieldBool,
		enumList: l,
	}
}

func NewEnumField(name string, field protoreflect.FieldDescriptor) *Field {
	enumDesc := field.Enum()
	values := enumDesc.Values()

	items := make([]list.Item, values.Len())
	for i := 0; i < values.Len(); i++ {
		enumVal := values.Get(i)
		items[i] = EnumItem{Name: string(enumVal.Name()), Value: string(enumVal.Number())}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(items, delegate, 30, 5)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)

	return &Field{
		name:     name,
		Kind:     FieldEnum,
		enumList: l,
	}
}

func ValidateInt(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a valid integer")
	}
	return nil
}

func ValidateUint(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a valid positive integer")
	}
	return nil
}

func ValidateFloat(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a valid number")
	}
	return nil
}
