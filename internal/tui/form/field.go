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
	path []string
	kind fieldKind

	textInput  textinput.Model
	enumPicker enumPicker

	validate func(string) error
}

func (f *Field) Depth() int {
	if len(f.path) == 0 {
		return 0
	}
	return len(f.path) - 1
}

func NewTextField(name string, path []string, placeholder string, charLimit int, validate func(string) error) *Field {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.Prompt = ""
	return &Field{
		name:      name,
		path:      path,
		kind:      FieldText,
		textInput: ti,
		validate:  validate,
	}
}

func NewBoolField(name string, path []string) *Field {
	items := []enumItem{
		{name: "false", value: "false"},
		{name: "true", value: "true"},
	}

	return &Field{
		name:       name,
		path:       path,
		kind:       FieldBool,
		enumPicker: newEnumPicker(items),
	}
}

func NewEnumField(name string, path []string, field protoreflect.FieldDescriptor) *Field {
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
		path:       path,
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
