// Package pparser provides functions and types for parsing files like
// /proc/status, /proc/pid/status and /proc/vmstat. Notably, the human-readable
// files which are lines of key-value pairs.
package pparser

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// NoUnknownFieldsFieldErr indicates that a field name didn't match the value
// in the struct.
type NoUnknownFieldsFieldErr struct {
	fieldName string
	value     string
}

func (n NoUnknownFieldsFieldErr) Error() string {
	return fmt.Sprintf("unrecognized field %q with value %q",
		n.fieldName, n.value)
}

// fieldIndex generates an index of field index to field-name, and the offset
// of the unknown fields field if present.
func fieldIndex(t interface{}) (map[string]int, int, reflect.Kind) {

	fieldIndex := map[string]int{}
	unknownIdx := -1
	unknownKind := reflect.Invalid

	objType := reflect.TypeOf(t)
	if objType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("concrete type must be passed to NewLineKVFileParser, got %s",
			objType))
	}
	for i := 0; i < objType.NumField(); i++ {

		field := objType.Field(i)
		if limitsTag, ok := field.Tag.Lookup("pparser"); ok {
			if limitsTag == "skip,unknown" {
				ftype := field.Type
				if ftype.Kind() != reflect.Map {
					continue
				}
				if ftype.Key().Kind() != reflect.String {
					continue
				}
				unknownIdx = i
				unknownKind = ftype.Elem().Kind()
				continue
			}
			// skip the fields labeled "skip"
			if limitsTag == "skip" {
				continue
			}
			fieldIndex[limitsTag] = i
		} else {
			fieldIndex[field.Name] = i
		}
	}

	return fieldIndex, unknownIdx, unknownKind

}

// NewLineKVFileParser constructs a new LineKVFileParser instance for the type
// passed as an argument. The UnknownFields field should be of type
// `map[string]int`, exported and have a `pparser:skip,unknown` struct field
// tag.
// Fields with the `pparser:"skip"` tag will be ignored. Any other value for
// the pparser field tag is interpreted as a preferred name for that field's key
// in the file.
// LineKVFileParser instances returned by NewLineKVFileParser contain an
// embedded index to make parsing a bit less inefficient. The `t` argument must
// be of the concrete struct-type, not a pointer to that type.
// Note: this is intended to be called once at startup for a type (usually
// within an `init()` func or as a package-level variable declaration).
func NewLineKVFileParser[T any](t T, splitKey string) *LineKVFileParser[T] {
	idx, unknownIdx, unknownKind := fieldIndex(t)

	return &LineKVFileParser[T]{
		idx:              idx,
		splitKey:         splitKey,
		unknownFieldsIdx: unknownIdx,
		unknownKind:      unknownKind,
		structType:       reflect.TypeOf(t),
	}

}

// LineKVFileParser provides a Parse(), it is not mutated by Parse(), and as
// such is thread-agnostic.
type LineKVFileParser[T any] struct {
	idx              map[string]int
	splitKey         string
	unknownFieldsIdx int
	unknownKind      reflect.Kind
	structType       reflect.Type
}

func trimStringWithMultiplier(s string) (string, int64) {
	if strings.HasSuffix(s, "kB") {
		return strings.TrimSpace(strings.TrimSuffix(s, "kB")), 1024
	}
	return s, 1
}

func (p *LineKVFileParser[T]) fieldKind(fieldName string) reflect.Kind {
	fieldIndex, knownField := p.idx[fieldName]
	if !knownField {
		return p.unknownKind
	}
	return p.structType.Field(fieldIndex).Type.Kind()
}

func (p *LineKVFileParser[T]) setIntField(
	outVal *reflect.Value, fieldName string, fieldValue int64) error {
	fieldIndex, knownField := p.idx[fieldName]
	var f reflect.Value
	if !knownField {
		if p.unknownFieldsIdx == -1 {
			panic("invariant failure: int-specific " +
				"function called with no field to handle it")
		}
		unknownFields := outVal.Field(p.unknownFieldsIdx)
		if unknownFields.IsNil() {
			unknownFields.Set(reflect.MakeMap(unknownFields.Type()))
		}
		insVal := reflect.New(unknownFields.Type().Elem()).Elem()
		if insVal.OverflowInt(fieldValue) {
			return fmt.Errorf(
				"unable to populate unknown field %q due to"+
					" overflow %d not representable by type %s",
				fieldName, fieldValue, insVal.Type().Kind())
		}
		insVal.SetInt(fieldValue)
		unknownFields.SetMapIndex(reflect.ValueOf(fieldName), insVal)

		return nil
	}
	f = outVal.Field(fieldIndex)
	if f.OverflowInt(fieldValue) {
		return fmt.Errorf(
			"unable to populate field %q due to"+
				" overflow %d not representable by type %s",
			fieldName, fieldValue, f.Type().Kind())
	}
	f.SetInt(fieldValue)

	return nil
}

func (p *LineKVFileParser[T]) setUintField(
	outVal *reflect.Value, fieldName string, fieldValue uint64) error {
	fieldIndex, knownField := p.idx[fieldName]
	var f reflect.Value
	if !knownField {
		if p.unknownFieldsIdx == -1 {
			panic("invariant failure: uint-specific " +
				"function called with no field to handle it")
		}
		unknownFields := outVal.Field(p.unknownFieldsIdx)
		if unknownFields.IsNil() {
			unknownFields.Set(reflect.MakeMap(unknownFields.Type()))
		}
		insVal := reflect.New(unknownFields.Type().Elem()).Elem()
		if insVal.OverflowUint(fieldValue) {
			return fmt.Errorf(
				"unable to populate unknown field %q due to"+
					" overflow %d not representable by type %s",
				fieldName, fieldValue, insVal.Type().Kind())
		}
		insVal.SetUint(fieldValue)
		unknownFields.SetMapIndex(reflect.ValueOf(fieldName), insVal)

		return nil
	}
	f = outVal.Field(fieldIndex)
	if f.OverflowUint(fieldValue) {
		return fmt.Errorf(
			"unable to populate field %q due to"+
				" overflow %d not representable by type %s",
			fieldName, fieldValue, f.Type().Kind())
	}
	f.SetUint(fieldValue)

	return nil
}

func (p *LineKVFileParser[T]) setFloatField(
	outVal *reflect.Value, fieldName string, fieldValue float64) error {
	fieldIndex, knownField := p.idx[fieldName]
	var f reflect.Value
	if !knownField {
		if p.unknownFieldsIdx == -1 {
			panic("invariant failure: int-specific " +
				"function called with no field to handle it")
		}
		unknownFields := outVal.Field(p.unknownFieldsIdx)
		if unknownFields.IsNil() {
			unknownFields.Set(reflect.MakeMap(unknownFields.Type()))
		}
		insVal := reflect.New(unknownFields.Type().Elem()).Elem()
		if insVal.OverflowFloat(fieldValue) {
			return fmt.Errorf(
				"unable to populate unknown field %q due to"+
					" overflow %g not representable by type %s",
				fieldName, fieldValue, insVal.Type().Kind())
		}
		insVal.SetFloat(fieldValue)
		unknownFields.SetMapIndex(reflect.ValueOf(fieldName), insVal)

		return nil
	}
	f = outVal.Field(fieldIndex)
	if f.OverflowFloat(fieldValue) {
		return fmt.Errorf(
			"unable to populate field %q due to"+
				" overflow %g not representable by type %s",
			fieldName, fieldValue, f.Type().Kind())
	}
	f.SetFloat(fieldValue)

	return nil
}
func (p *LineKVFileParser[T]) setStringField(
	outVal *reflect.Value, fieldName, fieldValue string) error {
	fieldIndex, knownField := p.idx[fieldName]
	var f reflect.Value
	if !knownField {
		if p.unknownFieldsIdx == -1 {
			panic("invariant failure: int-specific " +
				"function called with no field to handle it")
		}
		unknownFields := outVal.Field(p.unknownFieldsIdx)
		if unknownFields.IsNil() {
			unknownFields.Set(reflect.MakeMap(unknownFields.Type()))
		}
		insVal := reflect.New(unknownFields.Type().Elem()).Elem()
		insVal.SetString(fieldValue)
		unknownFields.SetMapIndex(reflect.ValueOf(fieldName), insVal)

		return nil
	}
	f = outVal.Field(fieldIndex)
	f.SetString(fieldValue)

	return nil
}

// Parse takes file-contents and an out-variable to populate. The out argument
// must be a pointer to the same type as passed to NewLineKVFileParser.
func (p *LineKVFileParser[T]) Parse(contentBytes []byte, out *T) error {
	outVal := reflect.ValueOf(out).Elem()

	b := bytes.NewBuffer(contentBytes)
	line, err := b.ReadString('\n')
	for ; len(line) > 0; line, err = b.ReadString('\n') {
		parts := strings.SplitN(line, p.splitKey, 2)
		if len(parts) < 2 {
			return fmt.Errorf("unable to split line %q", line)
		}

		trimmedVal := strings.TrimSpace(parts[1])

		k := p.fieldKind(parts[0])
		// Convert to the appropriate kind of value for the destination
		// field.
		switch k {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			{
				trimmedIntVal, mul := trimStringWithMultiplier(trimmedVal)
				val, intParseErr := strconv.ParseInt(trimmedIntVal, 10, 64)
				if intParseErr != nil {
					return fmt.Errorf("failed to parse line %q: %s",
						line, intParseErr)
				}
				val *= mul
				if setErr := p.setIntField(
					&outVal, parts[0], val); setErr != nil {
					return setErr
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			{
				trimmedUintVal, mul := trimStringWithMultiplier(trimmedVal)
				val, intParseErr := strconv.ParseUint(trimmedUintVal, 10, 64)
				if intParseErr != nil {
					return fmt.Errorf("failed to parse line %q: %s",
						line, intParseErr)
				}
				val *= uint64(mul)
				if setErr := p.setUintField(
					&outVal, parts[0], val); setErr != nil {
					return setErr
				}
			}
		case reflect.Float32, reflect.Float64:
			{
				trimmedFloatVal, mul := trimStringWithMultiplier(trimmedVal)
				val, floatParseErr := strconv.ParseFloat(trimmedFloatVal, 64)
				if floatParseErr != nil {
					return fmt.Errorf("failed to parse line %q: %s",
						line, floatParseErr)
				}
				val *= float64(mul)
				if setErr := p.setFloatField(
					&outVal, parts[0], val); setErr != nil {
					return setErr
				}
			}
		case reflect.String:
			if setErr := p.setStringField(
				&outVal, parts[0], trimmedVal); setErr != nil {
				return setErr
			}

		default:
			// TODO: implement slice and fixed-size array support
			return fmt.Errorf("unhandled field kind: %s", k)

		}
	}

	if err != io.EOF {
		return err
	}
	return nil
}
