package form

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fieldKind int

const (
	FieldText fieldKind = iota
	FieldBool
	FieldEnum
)

type Field struct {
	name string
	kind fieldKind

	textInput  textinput.Model // for text fields
	enumPicker enumPicker      // for enum and bool fields

	validate func(string) error
}

func NewTextField(name, placeholder string, charLimit int, validate func(string) error) *Field {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.Prompt = ""
	return &Field{
		name:      name,
		kind:      FieldText,
		textInput: ti,
		validate:  validate,
	}
}

func NewBoolField(name string) *Field {
	items := []enumItem{
		{name: "false", value: "false"},
		{name: "true", value: "true"},
	}

	return &Field{
		name:       name,
		kind:       FieldBool,
		enumPicker: newEnumPicker(items),
	}
}

func NewEnumField(name string, field protoreflect.FieldDescriptor) *Field {
	enumDesc := field.Enum()
	values := enumDesc.Values()

	items := make([]enumItem, values.Len())
	for i := 0; i < values.Len(); i++ {
		enumVal := values.Get(i)
		items[i] = enumItem{
			name:  string(enumVal.Name()),
			value: fmt.Sprintf("%d", enumVal.Number()),
		}
	}

	return &Field{
		name:       name,
		kind:       FieldEnum,
		enumPicker: newEnumPicker(items),
	}
}

func validateInt(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a valid integer")
	}
	return nil
}

func validateUint(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a valid positive integer")
	}
	return nil
}

func validateFloat(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a valid number")
	}
	return nil
}
