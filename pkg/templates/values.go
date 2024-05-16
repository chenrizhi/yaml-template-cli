package templates

import (
	"fmt"
	"github.com/imdario/mergo"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const GlobalKey = "global"

type Values map[string]interface{}

// YAML encodes the Values into a YAML string.
func (v Values) YAML() (string, error) {
	b, err := yaml.Marshal(v)
	return string(b), err
}

func (v Values) Table(name string) (Values, error) {
	table := v
	var err error

	for _, n := range parsePath(name) {
		if table, err = tableLookup(table, n); err != nil {
			break
		}
	}
	return table, err
}

func (v Values) AsMap() map[string]interface{} {
	if len(v) == 0 {
		return map[string]interface{}{}
	}
	return v
}

func (v Values) Encode(w io.Writer) error {
	out, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

func (v Values) OverrideValues(overrides Values) {
	err := mergo.Merge(&v, overrides, mergo.WithOverride)
	if err != nil {
		fmt.Println(err)
	}
}

func tableLookup(v Values, simple string) (Values, error) {
	v2, ok := v[simple]
	if !ok {
		return v, ErrNoTable{simple}
	}
	if vv, ok := v2.(map[string]interface{}); ok {
		return vv, nil
	}

	// This catches a case where a value is of type Values, but doesn't (for some
	// reason) match the map[string]interface{}. This has been observed in the
	// wild, and might be a result of a nil map of type Values.
	if vv, ok := v2.(Values); ok {
		return vv, nil
	}

	return Values{}, ErrNoTable{simple}
}

func ReadValues(data []byte) (vals Values, err error) {
	err = yaml.Unmarshal(data, &vals)
	if len(vals) == 0 {
		vals = Values{}
	}
	return vals, err
}

func ReadValuesFile(filename string) (Values, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return ReadValues(data)
}

func istable(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

func (v Values) PathValue(path string) (interface{}, error) {
	if path == "" {
		return nil, errors.New("YAML path cannot be empty")
	}
	return v.pathValue(parsePath(path))
}

func (v Values) pathValue(path []string) (interface{}, error) {
	if len(path) == 1 {
		// if exists must be root key not table
		if _, ok := v[path[0]]; ok && !istable(v[path[0]]) {
			return v[path[0]], nil
		}
		return nil, ErrNoValue{path[0]}
	}

	key, path := path[len(path)-1], path[:len(path)-1]
	// get our table for table path
	t, err := v.Table(joinPath(path...))
	if err != nil {
		return nil, ErrNoValue{key}
	}
	// check table for key and ensure value is not a table
	if k, ok := t[key]; ok && !istable(k) {
		return k, nil
	}
	return nil, ErrNoValue{key}
}

func parsePath(key string) []string { return strings.Split(key, ".") }

func joinPath(path ...string) string { return strings.Join(path, ".") }
