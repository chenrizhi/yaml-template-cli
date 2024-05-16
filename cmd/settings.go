package main

import (
	"github.com/spf13/pflag"
	"strings"
	"yaml-template-cli/pkg/templates"
)

var (
	overrides []string
)

type Settings struct {
	//Flags       *pflag.FlagSet
	Debug       bool
	OutputDir   string
	ValuesFiles []string
	InputDir    string
	Stdin       bool
	Overrides   templates.Values
}

func New() *Settings {
	return &Settings{}
}

func (s *Settings) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&s.OutputDir, "out", "o", s.OutputDir, "output directory")
	fs.StringSliceVarP(&s.ValuesFiles, "values", "v", []string{}, "values file path")
	fs.StringVarP(&s.InputDir, "in", "i", s.InputDir, "input directory")
	fs.BoolVarP(&s.Stdin, "stdin", "s", s.Stdin, "stdin")
	fs.StringSliceVarP(&overrides, "set", "", []string{}, "set")
}

func (s *Settings) ParseOverrideValues(overrides []string) {
	overridesMap := make(map[string]interface{})
	for _, override := range overrides {
		if len(override) == 0 {
			continue
		}
		split := strings.SplitN(override, "=", 2)
		if len(split) != 2 {
			continue
		}
		overridesMap[split[0]] = split[1]
	}
	s.Overrides = overridesMap
}
