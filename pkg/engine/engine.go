package engine

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"path"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"yaml-template-cli/pkg/templates"
)

const warnStartDelim = "ERR_START"
const warnEndDelim = "ERR_END"
const recursionMaxNums = 1000

var warnRegex = regexp.MustCompile(warnStartDelim + `((?s).*)` + warnEndDelim)

func warnWrap(warn string) string {
	return warnStartDelim + warn + warnEndDelim
}

type Engine struct {
	// If strict is enabled, templates rendering will fail if a templates references
	// a value that was not passed in.
	Strict bool
	// In LintMode, some 'required' templates values may be missing, so don't fail
	LintMode bool
}

func Render(tpl *templates.Template, values templates.Values) (map[string]string, error) {
	return new(Engine).Render(tpl, values)
}

func (e Engine) Render(tpl *templates.Template, values templates.Values) (map[string]string, error) {
	tmap := make(map[string]renderable)
	for _, file := range tpl.Templates {
		tmap[file.Name] = renderable{
			tpl:      string(file.Data),
			vals:     tpl.Values,
			basePath: "",
		}
	}
	return e.render(tmap)
}

// render takes a map of templates/values and renders them.
func (e Engine) render(tpls map[string]renderable) (rendered map[string]string, err error) {
	// Basically, what we do here is start with an empty parent templates and then
	// build up a list of templates -- one for each file. Once all of the templates
	// have been parsed, we loop through again and execute every templates.
	//
	// The idea with this process is to make it possible for more complex templates
	// to share common blocks, but to make the entire thing feel like a file-based
	// templates engine.
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("rendering templates failed: %v", r)
		}
	}()
	t := template.New("gotpl")
	if e.Strict {
		t.Option("missingkey=error")
	} else {
		// Not that zero will attempt to add default values for types it knows,
		// but will still emit <no value> for others. We mitigate that later.
		t.Option("missingkey=zero")
	}

	e.initFunMap(t)

	// We want to parse the templates in a predictable order. The order favors
	// higher-level (in file system) templates over deeply nested templates.
	keys := sortTemplates(tpls)

	for _, filename := range keys {
		r := tpls[filename]
		if _, err := t.New(filename).Parse(r.tpl); err != nil {
			return map[string]string{}, cleanupParseError(filename, err)
		}
	}

	rendered = make(map[string]string, len(keys))
	for _, filename := range keys {
		// Don't render partials. We don't care out the direct output of partials.
		// They are only included from other templates.
		if strings.HasPrefix(path.Base(filename), "_") {
			continue
		}
		// At render time, add information about the templates that is being rendered.
		vals := tpls[filename].vals
		vals["Template"] = templates.Values{"Name": filename, "BasePath": tpls[filename].basePath}
		var buf strings.Builder
		if err := t.ExecuteTemplate(&buf, filename, vals); err != nil {
			return map[string]string{}, cleanupExecError(filename, err)
		}

		// Work around the issue where Go will emit "<no value>" even if Options(missing=zero)
		// is set. Since missing=error will never get here, we do not need to handle
		// the Strict case.
		rendered[filename] = strings.ReplaceAll(buf.String(), "<no value>", "")
	}

	return rendered, nil
}

// initFunMap creates the Engine's FuncMap and adds context-specific functions.
func (e Engine) initFunMap(t *template.Template) {
	funcMap := funcMap()
	includedNames := make(map[string]int)

	// Add the templates-rendering functions here so we can close over t.
	funcMap["include"] = includeFun(t, includedNames)
	funcMap["tpl"] = tplFun(t, includedNames, e.Strict)

	// Add the `required` function here so we can use lintMode
	funcMap["required"] = func(warn string, val interface{}) (interface{}, error) {
		if val == nil {
			if e.LintMode {
				// Don't fail on missing required values when linting
				log.Printf("[INFO] Missing required value: %s", warn)
				return "", nil
			}
			return val, errors.Errorf(warnWrap(warn))
		} else if _, ok := val.(string); ok {
			if val == "" {
				if e.LintMode {
					// Don't fail on missing required values when linting
					log.Printf("[INFO] Missing required value: %s", warn)
					return "", nil
				}
				return val, errors.Errorf(warnWrap(warn))
			}
		}
		return val, nil
	}

	// Override sprig fail function for linting and wrapping message
	funcMap["fail"] = func(msg string) (string, error) {
		if e.LintMode {
			// Don't fail when linting
			log.Printf("[INFO] Fail: %s", msg)
			return "", nil
		}
		return "", errors.New(warnWrap(msg))
	}

	t.Funcs(funcMap)
}

// renderable is an object that can be rendered.
type renderable struct {
	// tpl is the current templates.
	tpl string
	// vals are the values to be supplied to the templates.
	vals templates.Values
	// namespace prefix to the templates of the current chart
	basePath string
}

// 'include' needs to be defined in the scope of a 'tpl' templates as
// well as regular file-loaded templates.
func includeFun(t *template.Template, includedNames map[string]int) func(string, interface{}) (string, error) {
	return func(name string, data interface{}) (string, error) {
		var buf strings.Builder
		if v, ok := includedNames[name]; ok {
			if v > recursionMaxNums {
				return "", errors.Wrapf(fmt.Errorf("unable to execute templates"), "rendering templates has a nested reference name: %s", name)
			}
			includedNames[name]++
		} else {
			includedNames[name] = 1
		}
		err := t.ExecuteTemplate(&buf, name, data)
		includedNames[name]--
		return buf.String(), err
	}
}

func sortTemplates(tpls map[string]renderable) []string {
	keys := make([]string, len(tpls))
	i := 0
	for key := range tpls {
		keys[i] = key
		i++
	}
	sort.Sort(sort.Reverse(byPathLen(keys)))
	return keys
}

func cleanupParseError(filename string, err error) error {
	tokens := strings.Split(err.Error(), ": ")
	if len(tokens) == 1 {
		// This might happen if a non-templating error occurs
		return fmt.Errorf("parse error in (%s): %s", filename, err)
	}
	// The first token is "templates"
	// The second token is either "filename:lineno" or "filename:lineNo:columnNo"
	location := tokens[1]
	// The remaining tokens make up a stacktrace-like chain, ending with the relevant error
	errMsg := tokens[len(tokens)-1]
	return fmt.Errorf("parse error at (%s): %s", string(location), errMsg)
}

func cleanupExecError(filename string, err error) error {
	if _, isExecError := err.(template.ExecError); !isExecError {
		return err
	}

	tokens := strings.SplitN(err.Error(), ": ", 3)
	if len(tokens) != 3 {
		// This might happen if a non-templating error occurs
		return fmt.Errorf("execution error in (%s): %s", filename, err)
	}

	// The first token is "templates"
	// The second token is either "filename:lineno" or "filename:lineNo:columnNo"
	location := tokens[1]

	parts := warnRegex.FindStringSubmatch(tokens[2])
	if len(parts) >= 2 {
		return fmt.Errorf("execution error at (%s): %s", string(location), parts[1])
	}

	return err
}

// As does 'tpl', so that nested calls to 'tpl' see the templates
// defined by their enclosing contexts.
func tplFun(parent *template.Template, includedNames map[string]int, strict bool) func(string, interface{}) (string, error) {
	return func(tpl string, vals interface{}) (string, error) {
		t, err := parent.Clone()
		if err != nil {
			return "", errors.Wrapf(err, "cannot clone templates")
		}

		// Re-inject the missingkey option, see text/templates issue https://github.com/golang/go/issues/43022
		// We have to go by strict from our engine configuration, as the option fields are private in Template.
		// TODO: Remove workaround (and the strict parameter) once we build only with golang versions with a fix.
		if strict {
			t.Option("missingkey=error")
		} else {
			t.Option("missingkey=zero")
		}

		// Re-inject 'include' so that it can close over our clone of t;
		// this lets any 'define's inside tpl be 'include'd.
		t.Funcs(template.FuncMap{
			"include": includeFun(t, includedNames),
			"tpl":     tplFun(t, includedNames, strict),
		})

		// We need a .New templates, as templates text which is just blanks
		// or comments after parsing out defines just addes new named
		// templates definitions without changing the main templates.
		// https://pkg.go.dev/text/template#Template.Parse
		// Use the parent's name for lack of a better way to identify the tpl
		// text string. (Maybe we could use a hash appended to the name?)
		t, err = t.New(parent.Name()).Parse(tpl)
		if err != nil {
			return "", errors.Wrapf(err, "cannot parse templates %q", tpl)
		}

		var buf strings.Builder
		if err := t.Execute(&buf, vals); err != nil {
			return "", errors.Wrapf(err, "error during tpl function execution for %q", tpl)
		}

		// See comment in renderWithReferences explaining the <no value> hack.
		return strings.ReplaceAll(buf.String(), "<no value>", ""), nil
	}
}

type byPathLen []string

func (p byPathLen) Len() int      { return len(p) }
func (p byPathLen) Swap(i, j int) { p[j], p[i] = p[i], p[j] }
func (p byPathLen) Less(i, j int) bool {
	a, b := p[i], p[j]
	ca, cb := strings.Count(a, "/"), strings.Count(b, "/")
	if ca == cb {
		return strings.Compare(a, b) == -1
	}
	return ca < cb
}
