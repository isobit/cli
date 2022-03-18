package opts

import (
	"fmt"
	"reflect"

	"github.com/huandu/xstrings"
	"github.com/pkg/errors"
)

type field struct {
	Name        string
	ShortName   string
	Help        string
	Placeholder string
	Required    bool
	EnvVarName  string
	HasArg      bool
	flagValue   *genericFlagValue
}

func (f field) Default() string {
	return f.flagValue.String()
}

func getFieldsFromConfig(config interface{}) ([]field, error) {
	configVal := reflect.ValueOf(config)
	if !configVal.IsValid() {
		return nil, errors.New("invalid config value")
	}
	if configVal.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("opts: config must be a struct pointer (got %s)", configVal.Type())
	}

	configElemVal := configVal.Elem()
	if !configElemVal.IsValid() {
		return nil, errors.New("invalid config element value")
	}
	if configElemVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("opts: config must be a struct pointer (got %s)", configVal.Type())
	}

	return getFields(configElemVal)
}

// sv must be a reflected struct pointer element
func getFields(sv reflect.Value) ([]field, error) {
	fields := []field{}
	for i := 0; i < sv.NumField(); i++ {
		sf := sv.Type().Field(i)
		val := sv.Field(i)

		// ignore unaddressable and unexported fields
		if !val.CanSet() {
			continue
		}

		meta := newFieldValueMeta(sf, val)

		// ignore fields with the "-" tag (like json)
		if _, ok := meta.tags["-"]; ok {
			continue
		}

		if meta.embedded {
			// embedded struct, recurse
			embeddedFields, err := getFields(val)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embeddedFields...)
		} else {
			field, err := getField(meta)
			if err != nil {
				return nil, errors.Wrapf(err, "problem with field %s.%s", sv.Type(), sf.Name)
			}
			fields = append(fields, field)
		}
	}
	return fields, nil
}

func getField(meta fieldValueMeta) (field, error) {
	f := field{}

	name, explicitName := meta.tags["name"]
	if !explicitName {
		name = xstrings.ToKebabCase(meta.structField.Name)
	}
	f.Name = name

	flagValue, err := getFlagValue(name, meta)
	if err != nil {
		return f, errors.Wrap(err, "not supported")
	}
	f.flagValue = flagValue

	f.Help = meta.tags["help"]
	f.Placeholder = meta.tags["placeholder"]
	// f.defaultString = meta.tags["default"]
	f.EnvVarName = meta.tags["env"]
	_, f.Required = meta.tags["required"]

	if shortName, ok := meta.tags["short"]; ok {
		if len(shortName) != 1 {
			return f, errors.New("short name must be 1 letter")
		}
		f.ShortName = shortName
	}

	f.HasArg = !flagValue.IsBoolFlag()

	return f, nil
}

type fieldValueMeta struct {
	structField reflect.StructField
	value       reflect.Value
	embedded    bool
	tags        map[string]string
}

func newFieldValueMeta(structField reflect.StructField, value reflect.Value) fieldValueMeta {
	tags := parseStructTagInner(structField.Tag.Get("opts"))
	return fieldValueMeta{
		structField: structField,
		value:       value,
		embedded:    structField.Anonymous,
		tags:        tags,
	}
}

func getFlagValue(name string, meta fieldValueMeta) (*genericFlagValue, error) {
	val := meta.value

	// Can't set into a nil pointer, so allocate a zero value for the field's
	// type to get a placeholder value to use with getters/stringers. Once
	// we've obtained a setter, we'll wrap it with pointerSetter so that the
	// actual pointer isn't set unless a flag is passed.
	isNilPointerSetter := false
	if val.Kind() == reflect.Ptr && val.IsZero() {
		val = reflect.New(meta.value.Type().Elem())
		isNilPointerSetter = true
	}

	var set setter
	var str stringer

	// Interfaces might be implemented using value or pointer receivers, so
	// we'll try both if we can take an address.
	interfaceables := []interface{}{val.Interface()}
	if val.CanAddr() {
		interfaceables = append(interfaceables, val.Addr().Interface())
	}
	for _, i := range interfaceables {
		set = tryGetSetter(i)
		str = tryGetStringer(i)
	}

	// override with tag-provided default stringer if available, otherwise fall
	// back on sprintfStringer if no stringer could be obtained from the
	// interfaceables
	if defaultString, ok := meta.tags["default"]; ok {
		str = staticStringer(defaultString)
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
			setter:      set,
			targetValue: meta.value,
			newValue:    val,
		}
	}

	return &genericFlagValue{
		name:       name,
		setter:     set,
		stringer:   str,
		isBoolFlag: meta.value.Kind() == reflect.Bool,
	}, nil
}

type setter interface {
	Set(s string) error
}

type pointerSetter struct {
	setter
	targetValue reflect.Value
	newValue    reflect.Value
}

func (ps pointerSetter) Set(s string) error {
	// Try to set the placeholder.
	err := ps.setter.Set(s)
	if err != nil {
		return err
	}

	// Set the target pointer to the placeholder pointer.
	ps.targetValue.Set(ps.newValue)

	return nil
}

type stringer interface {
	String() string
}

type genericFlagValue struct {
	name string
	setter
	stringer
	isBoolFlag bool
	setCount   uint
}

func (f *genericFlagValue) Set(s string) error {
	if f.setter == nil {
		panic("opts: genericFlagValue has no setter, this should not happen")
	}
	f.setCount += 1
	err := f.setter.Set(s)
	if err != nil {
		return errors.Wrapf(err, "failed to set option %s from string \"%s\"", f.name, s)
	}
	return nil
}

func (f *genericFlagValue) String() string {
	if f.stringer == nil {
		// Sometimes the flag package uses reflection to construct a zero
		// genericFlagValue, which obviously doesn't have a stringer, and then
		// calls String() on it to try to see if the default value is the zero
		// value. We don't care if it get the correct answer (it's only used in
		// PrintDefaults which we don't use).
		return "<unknown>"
	}
	return f.stringer.String()
}

func (f *genericFlagValue) IsBoolFlag() bool {
	return f.isBoolFlag
}
