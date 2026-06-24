package adm

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/yeqown/cassem/api/concept"
)

var requestValidator = newRequestValidator()

func newRequestValidator() *validator.Validate {
	validate := validator.New()
	validate.SetTagName("binding")
	if err := registerIdentifierValidator(validate); err != nil {
		panic(err)
	}

	return validate
}

func bindRequest(r *http.Request, obj any) error {
	if err := bindRequestURIParams(r, obj); err != nil {
		return err
	}
	if err := bindFormValues(obj, r.URL.Query(), "form"); err != nil {
		return err
	}
	if err := bindRequestJSON(r, obj); err != nil {
		return err
	}

	return requestValidator.Struct(obj)
}

func bindRequestURIParams(r *http.Request, obj any) error {
	values := make(url.Values)
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		for i, key := range routeCtx.URLParams.Keys {
			values.Set(key, routeCtx.URLParams.Values[i])
		}
	}

	return bindFormValues(obj, values, "uri")
}

func bindRequestJSON(r *http.Request, obj any) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(obj); err != nil && err != io.EOF {
		return err
	}

	return nil
}

func bindFormValues(obj any, values url.Values, tagName string) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("bind target must be a non-nil pointer")
	}
	return bindStructValue(v.Elem(), values, tagName)
}

func bindStructValue(v reflect.Value, values url.Values, tagName string) error {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)
		if structField.Anonymous {
			if err := bindStructValue(field, values, tagName); err != nil {
				return err
			}
			continue
		}

		name, defaultValue := parseBindTag(structField.Tag.Get(tagName))
		if name == "" || name == "-" {
			continue
		}
		rawValues, ok := values[name]
		if !ok && defaultValue != "" {
			rawValues = []string{defaultValue}
			ok = true
		}
		if !ok {
			continue
		}
		if err := setFieldValue(field, rawValues); err != nil {
			return fmt.Errorf("bind %s: %w", structField.Name, err)
		}
	}
	return nil
}

func parseBindTag(tag string) (name string, defaultValue string) {
	if tag == "" {
		return "", ""
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "default=") {
			defaultValue = strings.TrimPrefix(part, "default=")
		}
	}
	return name, defaultValue
}

func setFieldValue(field reflect.Value, values []string) error {
	if !field.CanSet() || len(values) == 0 {
		return nil
	}
	if field.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(field.Type(), 0, len(values))
		for _, item := range values {
			elem := reflect.New(field.Type().Elem()).Elem()
			if err := setScalarValue(elem, item); err != nil {
				return err
			}
			slice = reflect.Append(slice, elem)
		}
		field.Set(slice)
		return nil
	}
	return setScalarValue(field, values[len(values)-1])
}

func setScalarValue(field reflect.Value, value string) error {
	if field.CanAddr() {
		if unmarshaler, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(value))
		}
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	default:
		return fmt.Errorf("unsupported kind %s", field.Kind())
	}
	return nil
}

type contentTypeParam concept.ContentType

func (p *contentTypeParam) UnmarshalJSON(data []byte) error {
	var numeric int32
	if err := json.Unmarshal(data, &numeric); err == nil {
		*p = contentTypeParam(numeric)
		return nil
	}

	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	value, ok := concept.ContentType_value[strings.ToUpper(name)]
	if !ok {
		return fmt.Errorf("unknown contentType %q", name)
	}

	*p = contentTypeParam(value)
	return nil
}

func (p contentTypeParam) concept() concept.ContentType {
	return concept.ContentType(p)
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func isIdentifier(value string) bool {
	return identifierPattern.MatchString(value)
}

func registerIdentifierValidator(validate *validator.Validate) error {
	return validate.RegisterValidation("identifier", func(fl validator.FieldLevel) bool {
		return isIdentifier(fl.Field().String())
	})
}
