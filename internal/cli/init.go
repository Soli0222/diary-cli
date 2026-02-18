package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "設定ファイルを対話的に生成する",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "diary-cli")
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("⚠️  設定ファイルが既に存在します: %s\n", configPath)
		fmt.Print("上書きしますか？ (y/N) ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Println("中止しました")
				return nil
			}
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("📝 diary-cli の初期設定を行います")

	// Misskey settings
	fmt.Println("--- Misskey設定 ---")
	instanceURL := prompt(scanner, "MisskeyインスタンスURL", "https://misskey.io")
	token := prompt(scanner, "Misskeyアクセストークン", "")

	// Claude settings
	fmt.Println("\n--- Claude API設定 ---")
	apiKey := prompt(scanner, "Claude APIキー", "")
	model := prompt(scanner, "モデル", "claude-sonnet-4-6")

	// Diary settings
	fmt.Println("\n--- 日記設定 ---")
	outputDir := prompt(scanner, "日記の出力先ディレクトリ", "")
	author := prompt(scanner, "著者名", "Soli")
	editor := prompt(scanner, "エディタ", envOrDefault("EDITOR", "vim"))

	// Chat settings
	fmt.Println("\n--- 対話設定 ---")
	maxQ := prompt(scanner, "最大質問数", "8")
	minQ := prompt(scanner, "最低質問数", "3")

	// Build config YAML
	config := fmt.Sprintf(`# Misskey設定
misskey:
  instance_url: "%s"
  token: "%s"

# Claude API設定
claude:
  api_key: "%s"
  model: "%s"

# 日記設定
diary:
  output_dir: "%s"
  author: "%s"
  editor: "%s"

# 対話設定
chat:
  max_questions: %s
  min_questions: %s
`, instanceURL, token, apiKey, model, outputDir, author, editor, maxQ, minQ)

	// Create directory and write file
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\n✅ 設定ファイルを作成しました: %s\n", configPath)
	return nil
}

func prompt(scanner *bufio.Scanner, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}

	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			return input
		}
	}
	return defaultVal
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
