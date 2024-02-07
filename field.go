package cli

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/huandu/xstrings"
)

type field struct {
	Name        string
	ShortName   string
	Help        string
	Placeholder string
	Required    bool
	EnvVarName  string
	HasArg      bool
	Hidden      bool

	value *fieldValue
}

func (f field) Default() string {
	return f.value.String()
}

type argsField struct {
	setter func([]string)
}

func (cli *CLI) getFieldsFromConfig(config interface{}) ([]field, *argsField, error) {
	configVal := reflect.ValueOf(config)
	if !configVal.IsValid() {
		return nil, nil, fmt.Errorf("invalid config value")
	}
	if configVal.Kind() != reflect.Ptr {
		return nil, nil, fmt.Errorf("config must be a struct pointer (got %s)", configVal.Type())
	}

	configElemVal := configVal.Elem()
	if !configElemVal.IsValid() {
		return nil, nil, fmt.Errorf("invalid config element value")
	}
	if configElemVal.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("config must be a struct pointer (got %s)", configVal.Type())
	}

	return cli.getFields(configElemVal)
}

// sv must be a reflected struct pointer element
func (cli *CLI) getFields(sv reflect.Value) ([]field, *argsField, error) {
	fields := []field{}
	var argsField *argsField
	for i := 0; i < sv.NumField(); i++ {
		sf := sv.Type().Field(i)
		val := sv.Field(i)

		// ignore unaddressable and unexported fields
		if !val.CanSet() {
			continue
		}

		meta, err := newFieldValueMeta(sf, val)
		if err != nil {
			return nil, nil, fmt.Errorf("problem with field %s.%s: %w", sv.Type(), sf.Name, err)
		}

		// ignore fields with the "-" tag (like json)
		if meta.tags.exclude {
			continue
		}

		if meta.embedded || meta.tags.embed {
			// embedded struct, recurse
			embeddedFields, embeddedArgsField, err := cli.getFields(val)
			if err != nil {
				return nil, nil, err
			}
			fields = append(fields, embeddedFields...)
			if argsField == nil {
				argsField = embeddedArgsField
			}
		} else if meta.tags.args {
			field, err := cli.getArgsField(meta)
			if err != nil {
				return nil, nil, fmt.Errorf("problem with field %s.%s: %w", sv.Type(), sf.Name, err)
			}
			argsField = &field
		} else {
			field, err := cli.getField(meta)
			if err != nil {
				return nil, nil, fmt.Errorf("problem with field %s.%s: %w", sv.Type(), sf.Name, err)
			}
			fields = append(fields, field)
		}
	}
	return fields, argsField, nil
}

func (cli *CLI) getField(meta fieldValueMeta) (field, error) {
	name := meta.tags.name
	if name == "" {
		name = xstrings.ToKebabCase(meta.structField.Name)
	}

	fieldValue, err := cli.getFieldValue(name, meta)
	if err != nil {
		return field{}, fmt.Errorf("not supported: %w", err)
	}

	return field{
		Name:        name,
		ShortName:   meta.tags.short,
		Help:        meta.tags.help,
		Placeholder: meta.tags.placeholder,
		Required:    meta.tags.required,
		EnvVarName:  meta.tags.env,
		HasArg:      !fieldValue.isBoolFlag,
		Hidden:      meta.tags.hidden,
		value:       fieldValue,
	}, nil
}

func (cli *CLI) getArgsField(meta fieldValueMeta) (argsField, error) {
	val := meta.value
	if !val.CanAddr() {
		return argsField{}, fmt.Errorf("field has an args tag but type is not a slice of strings")
	}
	slicePointer, ok := val.Addr().Interface().(*[]string)
	if !ok {
		return argsField{}, fmt.Errorf("field has an args tag but type is not a slice of strings")
	}
	return argsField{
		setter: func(args []string) {
			*slicePointer = args
		},
	}, nil
}

type fieldValueMeta struct {
	structField reflect.StructField
	value       reflect.Value
	embedded    bool
	tags        fieldTags
}

func newFieldValueMeta(structField reflect.StructField, value reflect.Value) (fieldValueMeta, error) {
	tags, err := parseFieldTags(structField.Tag)
	if err != nil {
		return fieldValueMeta{}, err
	}

	meta := fieldValueMeta{
		structField: structField,
		value:       value,
		embedded:    structField.Anonymous,
		tags:        tags,
	}
	return meta, nil
}

type fieldTags struct {
	exclude       bool
	required      bool
	name          string
	short         string
	placeholder   string
	env           string
	help          string
	defaultString string
	hideDefault   bool
	hidden        bool
	append        bool
	embed         bool
	args          bool
}

func parseFieldTags(tag reflect.StructTag) (fieldTags, error) {
	t := fieldTags{}
	m := parseStructTagInner(tag.Get("cli"))
	pop := func(key string) (string, bool) {
		val, ok := m[key]
		if ok {
			delete(m, key)
		}
		return val, ok
	}

	if _, ok := pop("-"); ok {
		t.exclude = true
	}

	if _, ok := pop("required"); ok {
		t.required = true
	}

	if name, ok := pop("name"); ok {
		t.name = name
	}

	if short, ok := pop("short"); ok {
		if len(short) != 1 {
			return t, fmt.Errorf("short name must be 1 letter")
		}
		t.short = short
	}

	if placeholder, ok := pop("placeholder"); ok {
		t.placeholder = placeholder
	}

	if env, ok := pop("env"); ok {
		t.env = env
	}

	if help, ok := pop("help"); ok {
		t.help = help
	}

	if defaultString, ok := pop("default"); ok {
		t.defaultString = defaultString
		if defaultString == "" {
			t.hideDefault = true
		}
	}
	if _, ok := pop("nodefault"); ok {
		t.hideDefault = true
	}

	if _, ok := pop("append"); ok {
		t.append = true
	}

	if _, ok := pop("embed"); ok {
		t.embed = true
	}

	if _, ok := pop("hidden"); ok {
		t.hidden = true
	}

	if _, ok := pop("args"); ok {
		t.args = true
	}

	if len(m) > 0 {
		i := 0
		keys := make([]string, len(m))
		for k := range m {
			keys[i] = k
			i++
		}
		return t, fmt.Errorf("unknown tags: %s", strings.Join(keys, ", "))
	}

	return t, nil
}

func (cli *CLI) getFieldValue(name string, meta fieldValueMeta) (*fieldValue, error) {
	val := meta.value

	// Can't set into a nil pointer, so allocate a zero value for the field's
	// type to get a placeholder value to use with getters/stringers. Once
	// we've obtained a setter, we'll wrap it with pointerSetter so that the
	// actual pointer isn't set unless a flag is passed.
	isNilPointerSetter := false
	if val.Kind() == reflect.Ptr && val.IsZero() {
		val = reflect.New(val.Type().Elem())
		isNilPointerSetter = true
	}

	// If the field is repeatable, the value will be a slice, so create a
	// placeholder value of the element type. The setter for the placeholder
	// If the field has the "append" tag, the value will be a slice, so create
	// a placeholder value of the element type. The setter for the placeholder
	// will be wrapped with a repeatedSliceSetter later so that values are
	// appended to the target slice.
	shouldAppendSlice := false
	repeatableElemsArePointers := false
	if meta.tags.append {
		if val.Kind() != reflect.Slice {
			return nil, fmt.Errorf("field has append tag but value is not a slice type")
		}
		shouldAppendSlice = true
		valTypeElem := val.Type().Elem()
		if valTypeElem.Kind() == reflect.Ptr {
			repeatableElemsArePointers = true
			val = reflect.New(valTypeElem.Elem())
		} else {
			val = reflect.New(valTypeElem)
		}
	}

	var set Setter
	var str stringer

	// Interfaces might be implemented using value or pointer receivers, so
	// we'll try both if we can take an address.
	interfaceables := []interface{}{val.Interface()}
	if val.CanAddr() {
		interfaceables = append(interfaceables, val.Addr().Interface())
	}
	for _, i := range interfaceables {
		if set == nil && cli.Setter != nil {
			set = cli.Setter(i)
		}
		if set == nil {
			set = tryGetSetter(i)
		}
		if str == nil {
			str = tryGetStringer(i)
		}
	}

	// override with tag-provided default stringer if available, otherwise fall
	// back on sprintfStringer if no stringer could be obtained from the
	// interfaceables
	if meta.tags.defaultString != "" {
		str = staticStringer(meta.tags.defaultString)
	} else if meta.tags.hideDefault {
		str = staticStringer("")
	} else if str == nil {
		str = sprintfStringer{meta.value.Interface()}
	}

	if set == nil {
		return nil, fmt.Errorf("no setter for type %s", meta.value.Type())
	}
	if str == nil {
		return nil, fmt.Errorf("no stringer for type %s", meta.value.Type())
	}

	// Wrap nil pointer placeholder value setter with one that will set the
	// real pointer to the placeholder if the flag is passed.
	if isNilPointerSetter {
		set = pointerSetter{
			setter:           set,
			targetValue:      meta.value,
			placeholderValue: val,
		}
	}

	// Wrap element placeholder setter with one that will append to the real
	// value slice when the flag is passed.
	if shouldAppendSlice {
		set = appendSliceSetter{
			setter:           set,
			targetValue:      meta.value,
			placeholderValue: val,
			elemsArePointers: repeatableElemsArePointers,
		}
	}

	return &fieldValue{
		Setter:     set,
		stringer:   str,
		isBoolFlag: meta.value.Kind() == reflect.Bool,
	}, nil
}

type Setter interface {
	Set(s string) error
}

type pointerSetter struct {
	setter           Setter
	targetValue      reflect.Value
	placeholderValue reflect.Value
}

func (ps pointerSetter) Set(s string) error {
	// Try to set the placeholder.
	if err := ps.setter.Set(s); err != nil {
		return err
	}

	// Set the target pointer to the placeholder pointer.
	ps.targetValue.Set(ps.placeholderValue)

	return nil
}

type appendSliceSetter struct {
	setter           Setter
	targetValue      reflect.Value
	placeholderValue reflect.Value
	elemsArePointers bool
}

func (rss appendSliceSetter) Set(s string) error {
	// Try to set the placeholder.
	if err := rss.setter.Set(s); err != nil {
		return err
	}

	// Append the placeholder to the target slice.
	newElem := rss.placeholderValue.Elem()
	if rss.elemsArePointers {
		tmp := reflect.New(newElem.Type())
		tmp.Elem().Set(newElem)
		newElem = tmp
	}
	rss.targetValue.Set(reflect.Append(rss.targetValue, newElem))

	return nil
}

type stringer interface {
	String() string
}

type fieldValue struct {
	Setter
	stringer
	isBoolFlag bool
	setCount   uint
}

func (f *fieldValue) Set(s string) error {
	if f.Setter == nil {
		panic("cli: fieldValue has no setter, this should not happen")
	}
	f.setCount += 1
	if err := f.Setter.Set(s); err != nil {
		return err
	}
	return nil
}
