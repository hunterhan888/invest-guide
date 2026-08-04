package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnvFromCwd 从当前工作目录向上逐级查找 .env 并加载（不覆盖已存在的环境变量）。
// 在 backend/ 下运行时，能找到项目根的 .env（../.env）。
func loadDotEnvFromCwd() error {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	for {
		path := filepath.Join(dir, ".env")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return loadDotEnvFile(path)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// loadDotEnvFile 解析一个 .env 文件并写入环境变量。
// 仅当变量未设置时写入（不覆盖已存在的环境变量，与 godotenv.Load 语义一致）。
// 支持：空行、以 # 开头的注释行、KEY=VALUE、可选单双引号包裹的值。
func loadDotEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}
