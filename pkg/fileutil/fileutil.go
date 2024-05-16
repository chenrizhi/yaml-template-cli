package fileutil

import (
	"github.com/imdario/mergo"
	"io"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"yaml-template-cli/pkg/templates"
)

func ReadFile(filename string) ([]byte, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	return os.WriteFile(filename, data, perm)
}

func ReadTemplateFiles(tplFiles, valuesFiles []string) (*templates.Template, error) {
	tpls := &templates.Template{
		Templates: []templates.File{},
		Values:    templates.Values{},
	}
	for _, file := range tplFiles {
		// 读取文件内容
		data, err := ReadFile(file)
		if err != nil {
			return nil, err
		}
		// 添加到模板中
		tpls.Templates = append(tpls.Templates, templates.File{
			Name: file,
			Data: data,
		})
	}
	values, err := ReadValuesFiles(valuesFiles)
	if err != nil {
		return nil, err
	}
	tpls.Values = values
	return tpls, nil
}

func ReadValuesFiles(files []string) (templates.Values, error) {
	values := map[string]interface{}{}
	for _, file := range files {
		// 读取文件内容
		data, err := ReadFile(file)
		if err != nil {
			return nil, err
		}
		// 解析文件内容为YAML
		yamlData := map[string]interface{}{}
		if err := yaml.Unmarshal(data, &yamlData); err != nil {
			return nil, err
		}
		// 合并到values
		err = mergo.Merge(&values, yamlData, mergo.WithOverride)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

// ListAllFilesWithExt
// 返回指定路径下所有指定扩展名的文件
// 暂不支持递归遍历子目录
func ListAllFilesWithExt(path string, exts []string) ([]string, error) {
	var tplFiles []string
	// 读取文件夹下的所有文件
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	// 遍历文件
	for _, file := range files {
		// 判断是否为文件
		if file.IsDir() {
			continue
		}
		// 判断扩展名是否符合条件
		if contains(exts, filepath.Ext(file.Name())) {
			tplFiles = append(tplFiles, filepath.Join(path, file.Name()))
		}
	}
	return tplFiles, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
