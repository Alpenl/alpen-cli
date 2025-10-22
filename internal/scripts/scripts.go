package scripts

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpen/alpen-cli/internal/config"
)

// ResolveRoot 返回 ~/.alpen/config/scripts 目录
func ResolveRoot() (string, error) {
	root, err := config.DefaultScriptsRoot()
	if err != nil {
		return "", err
	}
	return filepath.Clean(root), nil
}

// ResolveCommandTarget 根据命令 token 解析真实脚本路径
func ResolveCommandTarget(token string, workingDir string) (string, bool, error) {
	if token == "" {
		return "", false, nil
	}
	if filepath.IsAbs(token) {
		return token, true, nil
	}
	if strings.HasPrefix(token, "./") || strings.HasPrefix(token, "../") {
		base := workingDir
		if base == "" {
			var err error
			base, err = os.Getwd()
			if err != nil {
				return "", false, err
			}
		}
		return filepath.Join(base, token), true, nil
	}
	return "", false, nil
}

// VerifyExecutable 校验脚本文件是否存在、可执行且包含 Shebang
func VerifyExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("脚本 %s 不存在", path)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("脚本 %s 指向目录", path)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("脚本 %s 缺少可执行权限", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	head, err := reader.Peek(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(head) < 2 || head[0] != '#' || head[1] != '!' {
		return fmt.Errorf("脚本 %s 缺少 Shebang", path)
	}
	return nil
}

// IsUnderRoot 判断脚本是否位于根目录下
func IsUnderRoot(path string, root string) bool {
	if root == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != "" && rel != "." && !strings.HasPrefix(rel, "..")
}
