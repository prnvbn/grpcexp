package form

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fieldKind int

const (
	FieldText fieldKind = iota
	FieldBool
	FieldEnum
	FieldSubmit
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

func NewFieldFromProto(field protoreflect.FieldDescriptor, path []string) *Field {
	name := string(field.Name())
	displayName := strings.Join(path, ".")

	switch field.Kind() {
	case protoreflect.StringKind:
		return NewTextField(displayName, path, fmt.Sprintf("Enter %s...", name), 256, nil)

	case protoreflect.BoolKind:
		return NewBoolField(displayName, path)

	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return NewTextField(displayName, path, "Enter integer...", 64, validateInt)

	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return NewTextField(displayName, path, "Enter positive integer...", 64, validateUint)

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return NewTextField(displayName, path, "Enter number...", 64, validateFloat)

	case protoreflect.EnumKind:
		return NewEnumField(displayName, path, field)

	case protoreflect.BytesKind:
		return NewTextField(displayName, path, "Enter hex bytes (e.g., deadbeef)...", 512, nil)

	default:
		return nil
	}
}
