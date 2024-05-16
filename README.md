# yaml-template-cli

类似于helm模板那样，使用yaml文件作为模板，使用命令行参数或指定values.yaml作为变量，生成新的yaml文件。

## Usage

参数如下：

> -i              输入目录
> 
> -o              输出目录
> 
> -s              使用标准输入（stdin）作为模板
> 
> -v              指定values文件路径，可以指定多个，多个values会merge成一个，后者覆盖前着
> 
> --set key=value 命令行设置value，优先级最高，会覆盖values文件的值

常见用法：

- 简单的调试

  ```bash
  echo "name: {{.name}}" | yaml-template-cli -s --set name=haha
  ```


- 完整的用法

  ```bash
  yaml-template-cli -i example -o out -v values-dev.yaml
  ```

  这里例子中，将example目录作为输入源，out目录作为输出，命令执行完成后，会在out目录下生成所有渲染好的文件。
.

- 结果输出到终端

  如果你需要将结果输出到终端，而不是生成文件，可以省略`-o`参数

  ```bash
  yaml-template-cli -i example -v values-dev.yaml
  ```