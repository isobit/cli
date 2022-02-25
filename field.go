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

		meta := NewFieldValueMeta(sf, val)

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

	flagValue, err := getFlagValue(meta)
	if err != nil {
		return f, errors.Wrap(err, "not supported")
	}
	f.flagValue = flagValue

	name, explicitName := meta.tags["name"]
	if !explicitName {
		name = xstrings.ToKebabCase(meta.structField.Name)
	}
	f.Name = name

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

func NewFieldValueMeta(structField reflect.StructField, value reflect.Value) fieldValueMeta {
	tags := parseStructTagInner(structField.Tag.Get("opts"))
	if helpTag, ok := structField.Tag.Lookup("help"); ok {
		tags["help"] = helpTag
	}

	return fieldValueMeta{
		structField: structField,
		value:       value,
		embedded:    structField.Anonymous,
		tags:        tags,
	}
}

func getFlagValue(meta fieldValueMeta) (*genericFlagValue, error) {
	// if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
	// 	fmt.Printf("%s %+v %+v\n", meta.structField.Name, val.Type(), val.Kind())
	// }

	val := meta.value
	isNilPointerSetter := false
	if val.Kind() == reflect.Ptr && val.IsZero() {
		// fmt.Printf("nil pointer %s %+v %+v\n", meta.structField.Name, meta.value.Type(), meta.value.Kind())
		val = reflect.New(meta.value.Type().Elem())
		isNilPointerSetter = true
	}

	var set setter
	var str stringer

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

	if isNilPointerSetter {
		set = pointerSetter{
			setter:      set,
			targetValue: meta.value,
			newValue:    val,
		}
	}

	return &genericFlagValue{
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
	ps.setter.Set(s)
	ps.targetValue.Set(ps.newValue)
	return nil
}

type stringer interface {
	String() string
}

type genericFlagValue struct {
	setter
	stringer
	isBoolFlag bool
	setCount   uint
}

func (f *genericFlagValue) Set(s string) error {
	if f.setter == nil {
		panic("genericFlagValue has no setter, this should not happen")
	}
	f.setCount += 1
	return f.setter.Set(s)
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
