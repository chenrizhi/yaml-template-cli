package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"yaml-template-cli/pkg/engine"
	"yaml-template-cli/pkg/fileutil"
	"yaml-template-cli/pkg/templates"
)

var globalUsage = `The YAML templates renderer
`

var settings = New()

func init() {
	log.SetFlags(log.Lshortfile)
}

func NewRootCmd(out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "yaml-templates-cli",
		Short:        "The YAML templates renderer",
		Long:         globalUsage,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			err := handler()
			if err != nil {
				if _, err := out.Write([]byte(err.Error())); err != nil {
					return
				}
			}
		},
	}
	flags := cmd.PersistentFlags()

	settings.AddFlags(flags)

	flags.ParseErrorsWhitelist.UnknownFlags = true
	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}
	os.Args = args[:1] // 这里解析参数，将参数去掉了，避免重复处理

	cmd.AddCommand(versionCmd)

	settings.ParseOverrideValues(overrides)

	return cmd, nil
}

func handler() error {
	if settings.Stdin {
		in, _ := io.ReadAll(os.Stdin)
		values, err := fileutil.ReadValuesFiles(settings.ValuesFiles)
		if err != nil {
			return err
		}
		values.OverrideValues(settings.Overrides)
		tpl := &templates.Template{
			Templates: []templates.File{
				{
					Name: "stdin",
					Data: in,
				},
			},
			Values: values,
		}
		render, err := engine.Render(tpl, tpl.Values)
		if err != nil {
			return err
		}
		if settings.OutputDir == "" {
			for k, v := range render {
				fmt.Printf("# Source: %s\n%s\n---\n", k, v)
			}
		}
		return nil
	}
	if settings.InputDir == "" {
		return fmt.Errorf("input dir is not specified")
	}
	yamlFiles, err := fileutil.ListAllFilesWithExt(settings.InputDir, []string{".yaml", ".yml"})
	if err != nil {
		return err
	}
	tpls, err := fileutil.ReadTemplateFiles(yamlFiles, settings.ValuesFiles)
	if err != nil {
		return err
	}
	tpls.Values.OverrideValues(settings.Overrides)
	render, err := engine.Render(tpls, tpls.Values)
	if err != nil {
		return err
	}
	if settings.OutputDir == "" {
		for k, v := range render {
			fmt.Printf("# Source: %s\n%s\n---\n", k, v)
		}
		return nil
	}
	for k, v := range render {
		err := fileutil.WriteFile(filepath.Join(settings.OutputDir, path.Base(k)), []byte(v), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
