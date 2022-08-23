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
	EnvKey      string
	HasArg      bool
	Hidden      bool
	flagValue   *genericFlagValue
}

func (f field) Default() string {
	return f.flagValue.String()
}

func (cli *CLI) getFieldsFromConfig(config interface{}) ([]field, error) {
	configVal := reflect.ValueOf(config)
	if !configVal.IsValid() {
		return nil, fmt.Errorf("invalid config value")
	}
	if configVal.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("config must be a struct pointer (got %s)", configVal.Type())
	}

	configElemVal := configVal.Elem()
	if !configElemVal.IsValid() {
		return nil, fmt.Errorf("invalid config element value")
	}
	if configElemVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config must be a struct pointer (got %s)", configVal.Type())
	}

	return cli.getFields(configElemVal)
}

// sv must be a reflected struct pointer element
func (cli *CLI) getFields(sv reflect.Value) ([]field, error) {
	fields := []field{}
	for i := 0; i < sv.NumField(); i++ {
		sf := sv.Type().Field(i)
		val := sv.Field(i)

		// ignore unaddressable and unexported fields
		if !val.CanSet() {
			continue
		}

		meta, err := newFieldValueMeta(sf, val)
		if err != nil {
			return nil, fmt.Errorf("problem with field %s.%s: %w", sv.Type(), sf.Name, err)
		}

		// ignore fields with the "-" tag (like json)
		if meta.tags.exclude {
			continue
		}

		if meta.embedded {
			// embedded struct, recurse
			embeddedFields, err := cli.getFields(val)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embeddedFields...)
		} else {
			field, err := cli.getField(meta)
			if err != nil {
				return nil, fmt.Errorf("problem with field %s.%s: %w", sv.Type(), sf.Name, err)
			}
			fields = append(fields, field)
		}
	}
	return fields, nil
}

func (cli *CLI) getField(meta fieldValueMeta) (field, error) {
	name := meta.tags.name
	if name == "" {
		name = xstrings.ToKebabCase(meta.structField.Name)
	}

	flagValue, err := cli.getFlagValue(name, meta)
	if err != nil {
		return field{}, fmt.Errorf("not supported: %w", err)
	}

	f := field{
		Name:        name,
		ShortName:   meta.tags.short,
		Help:        meta.tags.help,
		Placeholder: meta.tags.placeholder,
		Required:    meta.tags.required,
		EnvKey:      meta.tags.env,
		HasArg:      !flagValue.IsBoolFlag(),
		Hidden:      meta.tags.hidden,
		flagValue:   flagValue,
	}
	return f, nil
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
	increment     bool
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

	if _, ok := pop("hidden"); ok {
		t.hidden = true
	}

	if _, ok := pop("append"); ok {
		t.append = true
	}

	if _, ok := pop("increment"); ok {
		if t.append {
			return t, fmt.Errorf("tags are mutually exclusive: append, increment")
		}
		t.increment = true
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

func (cli *CLI) getFlagValue(name string, meta fieldValueMeta) (*genericFlagValue, error) {
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

	// If the field has the "append" tag, the value will be a slice, so create
	// a placeholder value of the element type. The setter for the placeholder
	// will be wrapped with a repeatedSliceSetter later so that values are
	// appended to the target slice.
	repeatableElemsArePointers := false
	if meta.tags.append {
		if val.Kind() != reflect.Slice {
			return nil, fmt.Errorf("field has append tag but value is not a slice type")
		}
		valTypeElem := val.Type().Elem()
		if valTypeElem.Kind() == reflect.Ptr {
			repeatableElemsArePointers = true
			val = reflect.New(valTypeElem.Elem())
		} else {
			val = reflect.New(valTypeElem)
		}
	}

	// Interfaces might be implemented using value or pointer receivers, so
	// we'll try both if we can take an address.
	interfaceables := []interface{}{val.Interface()}
	if val.CanAddr() {
		interfaceables = append(interfaceables, val.Addr().Interface())
	}

	// Try to obtain a setter from the interfacables, using the cli Setter
	// function first if non-nil.
	set := cli.tryGetSetter(interfaceables)
	// var set Setter
	// for _, i := range interfaceables {
	// 	if cli.Setter != nil {
	// 		set = cli.Setter(i)
	// 		if set != nil {
	// 			break
	// 		}
	// 	}
	// 	set = tryGetSetter(i)
	// 	if set != nil {
	// 		break
	// 	}
	// }
	if set == nil {
		return nil, fmt.Errorf("no setter for type %s", meta.value.Type())
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
	if meta.tags.append {
		set = appendSliceSetter{
			setter:           set,
			targetValue:      meta.value,
			placeholderValue: val,
			elemsArePointers: repeatableElemsArePointers,
		}
	}

	// Use tag-provided default stringer if available, otherwise fall back on
	// sprintfStringer if no stringer could be obtained from the
	// interfaceables.
	var str stringer
	if meta.tags.defaultString != "" {
		str = staticStringer(meta.tags.defaultString)
	} else if meta.tags.hideDefault {
		str = staticStringer("")
	} else {
		for _, i := range interfaceables {
			str = tryGetStringer(i)
			if str != nil {
				break
			}
		}
		if str == nil {
			str = sprintfStringer{meta.value.Interface()}
		}
	}
	if str == nil {
		return nil, fmt.Errorf("no stringer for type %s", meta.value.Type())
	}

	return &genericFlagValue{
		name:       name,
		Setter:     set,
		stringer:   str,
		isBoolFlag: meta.value.Kind() == reflect.Bool,
	}, nil
}
func (cli *CLI) tryGetSetter(interfaceables []interface{}) Setter {
	for _, i := range interfaceables {
		if cli.Setter != nil {
			if set := cli.Setter(i); set != nil {
				return set
			}
		}
		if set := tryGetSetter(i); set != nil {
			return set
		}
	}
	return nil
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

type genericFlagValue struct {
	name string
	Setter
	stringer
	isBoolFlag bool
	setCount   uint
}

func (f *genericFlagValue) Set(s string) error {
	if f.Setter == nil {
		panic("cli: genericFlagValue has no setter, this should not happen")
	}
	f.setCount += 1
	if err := f.Setter.Set(s); err != nil {
		return err
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
