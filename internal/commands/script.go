package commands

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/scripts"
	"github.com/alpen/alpen-cli/internal/ui"
)

// NewScriptCommand 创建 script 子命令
func NewScriptCommand(_ Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "script",
		Short:         "脚本仓库辅助工具",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newScriptListCommand())
	cmd.AddCommand(newScriptDoctorCommand())
	return cmd
}

func newScriptListCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "ls",
		Short:         "列出脚本仓库中的脚本",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := scriptsRootPath()
			if err != nil {
				return err
			}
			writer := cmd.OutOrStdout()
			ui.KeyValue(writer, "脚本目录", root)

			var scripts []string
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(root, path)
				scripts = append(scripts, rel)
				return nil
			})
			if errors.Is(err, os.ErrNotExist) {
				ui.Warning(writer, "尚未创建脚本目录，可执行 %s 初始化", ui.Highlight("alpen init"))
				return nil
			}
			if err != nil {
				return err
			}
			if len(scripts) == 0 {
				ui.Info(writer, "当前目录下没有脚本文件")
				return nil
			}
			for _, s := range scripts {
				fmt.Fprintf(writer, "  %s\n", s)
			}
			return nil
		},
	}
}

func newScriptDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "doctor",
		Short:         "检查脚本仓库安全性",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := scriptsRootPath()
			if err != nil {
				return err
			}
			writer := cmd.OutOrStdout()
			ui.KeyValue(writer, "脚本目录", root)

			var issues int
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(root, path)
				if info.Mode()&0o111 == 0 {
					issues++
					ui.Warning(writer, "脚本缺少可执行权限: %s", rel)
				}
				if err := checkShebang(path); err != nil {
					issues++
					ui.Warning(writer, "脚本缺少 Shebang: %s (%v)", rel, err)
				}
				return nil
			})
			if errors.Is(err, os.ErrNotExist) {
				ui.Warning(writer, "尚未创建脚本目录，可执行 %s 初始化", ui.Highlight("alpen init"))
				return nil
			}
			if err != nil {
				return err
			}
			if issues == 0 {
				ui.Success(writer, "脚本仓库检查通过")
			} else {
				ui.Info(writer, "共发现 %d 个可改进项", issues)
			}
			return nil
		},
	}
}

func scriptsRootPath() (string, error) {
	root, err := scripts.ResolveRoot()
	if err != nil {
		return "", err
	}
	return root, nil
}

func checkShebang(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 3)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if n < 2 || buf[0] != '#' || buf[1] != '!' {
		return fmt.Errorf("缺少 Shebang")
	}
	return nil
}
