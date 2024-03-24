package protocol

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Message struct {
	Name string
	Raw  string
}

func NewMessage(name, raw string) *Message {
	return &Message{
		Name: name,
		Raw:  raw,
	}
}

func EncodeMessage(message any) ([]byte, error) {
	rv := reflect.ValueOf(message)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return []byte(""), fmt.Errorf("nil pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return []byte(""), fmt.Errorf("encode expects a struct, got %s", rv.Kind())
	}

	rt := rv.Type()
	var sb strings.Builder

	sb.WriteString(strings.ToLower(rt.Name()))

	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		key := sf.Tag.Get("teamspeak")
		if key == "" {
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}

			fv = fv.Elem()
		}

		var val string
		switch fv.Kind() {
		case reflect.String:
			val = fv.String()
		case reflect.Slice:
			if fv.Type().Elem().Kind() != reflect.Uint8 {
				return []byte(""), fmt.Errorf("field %s: unsupported Slice kind %s", sf.Name, fv.Type().Elem().Kind())
			}
			val = base64.StdEncoding.EncodeToString(fv.Bytes())
		case reflect.Bool:
			val = map[bool]string{true: "1", false: "0"}[fv.Bool()]
		case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
			val = fmt.Sprintf("%d", fv.Uint())
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
			val = fmt.Sprintf("%d", fv.Int())
		default:
			return []byte(""), fmt.Errorf("field %s: unsupported kind %s", sf.Name, fv.Kind())
		}
		sb.WriteByte(' ')
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(EscapeTS3(val))
	}
	return []byte(sb.String()), nil
}

func DecodeMessage(message string, out any) error {
	groups := map[string]string{}
	fields := strings.Fields(message)

	for _, f := range fields[1:] {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) == 2 {
			groups[parts[0]] = UnescapeTS3(parts[1])
		} else if len(parts) == 1 {
			groups[parts[0]] = ""
		}
	}

	rv := reflect.ValueOf(out)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return fmt.Errorf("nil pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("decode expects a struct, got %s", rv.Kind())
	}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)

		key := sf.Tag.Get("teamspeak")
		if key == "" {
			continue
		}

		val, ok := groups[key]
		if !ok {
			continue
		}

		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(val)
		case reflect.Int:
			var err error
			val, err := strconv.ParseInt(val, 10, 32)
			if err != nil {
				return err
			}
			fv.SetInt(val)
		case reflect.Uint16:
			var err error
			val, err := strconv.ParseUint(val, 10, 16)
			if err != nil {
				return err
			}
			fv.SetUint(val)
		default:
			return fmt.Errorf("unsupported field type %s", fv.Kind())
		}
	}
	return nil
}

func ParseMessageName(s string) (string, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", fmt.Errorf("invalid message: %q", s)
	}
	return fields[0], nil
}

func ParseMessage(s string) (*Message, error) {
	name, err := ParseMessageName(s)
	if err != nil {
		return nil, err
	}
	return NewMessage(name, s), nil
}

type KV struct {
	Key string
	Val string
}

func BuildCommand(name string, kvs []KV) []byte {
	var b strings.Builder
	b.WriteString(name)
	for _, kv := range kvs {
		b.WriteByte(' ')
		b.WriteString(kv.Key)
		b.WriteByte('=')
		b.WriteString(kv.Val)
	}
	return []byte(b.String())
}

// EscapeTS3 encodes TS3 query escape sequences.
func EscapeTS3(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ':
			b.WriteString(`\s`)
		case '|':
			b.WriteString(`\p`)
		case '/':
			b.WriteString(`\/`)
		case '\\':
			b.WriteString(`\\`)
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\v':
			b.WriteString(`\v`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// UnescapeTS3 decodes TS3 query escape sequences.
func UnescapeTS3(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch != '\\' || i+1 >= len(s) {
			b.WriteByte(ch)
			continue
		}
		i++
		switch s[i] {
		case 's':
			b.WriteByte(' ')
		case 'p':
			b.WriteByte('|')
		case '/':
			b.WriteByte('/')
		case '\\':
			b.WriteByte('\\')
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'v':
			b.WriteByte('\v')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
