package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/config"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "設定ファイルを対話的に生成する",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	configDir, err := config.DefaultConfigDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("設定ファイルが既に存在します: %s\n", configPath)
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
	fmt.Println("diary-cli の初期設定を行います")

	fmt.Println("\n[Misskey]")
	instanceURL := prompt(scanner, "MisskeyインスタンスURL", "https://misskey.io")
	token := prompt(scanner, "Misskeyアクセストークン", "")

	fmt.Println("\n[AI]")
	defaultProvider := prompt(scanner, "デフォルトAIプロバイダ", "claude")

	fmt.Println("\n[Claude]")
	claudeAPIKey := prompt(scanner, "Claude APIキー", "")
	claudeModel := prompt(scanner, "Claudeモデル", "claude-sonnet-4-6")

	fmt.Println("\n[OpenAI]")
	openAIAPIKey := prompt(scanner, "OpenAI APIキー", "")
	openAIModel := prompt(scanner, "OpenAIモデル", "gpt-5.4-mini")

	fmt.Println("\n[Gemini]")
	geminiAPIKey := prompt(scanner, "Gemini APIキー", "")
	geminiModel := prompt(scanner, "Geminiモデル", "gemini-3.1-flash-preview")

	fmt.Println("\n[Diary]")
	outputDir := prompt(scanner, "出力先ディレクトリ", "./diary")
	author := prompt(scanner, "author", config.EnvOrDefault("USER", "Soli"))
	editor := prompt(scanner, "editor", config.EnvOrDefault("EDITOR", "vim"))
	timezone := prompt(scanner, "timezone", "Asia/Tokyo")

	fmt.Println("\n[Summaly]")
	summalyEndpoint := prompt(scanner, "Summalyエンドポイント (任意)", "")

	fmt.Println("\n[Discord]")
	webhookURL := prompt(scanner, "Discord Webhook URL (任意)", "")

	content := fmt.Sprintf(`misskey:
  instance_url: "%s"
  token: "%s"

ai:
  default_provider: "%s"
  claude:
    api_key: "%s"
    model: "%s"
  openai:
    api_key: "%s"
    model: "%s"
  gemini:
    api_key: "%s"
    model: "%s"

diary:
  output_dir: "%s"
  author: "%s"
  editor: "%s"
  timezone: "%s"

summaly:
  endpoint: "%s"

discord:
  webhook_url: "%s"
`, instanceURL, token, defaultProvider, claudeAPIKey, claudeModel, openAIAPIKey, openAIModel, geminiAPIKey, geminiModel, outputDir, author, editor, timezone, summalyEndpoint, webhookURL)

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("設定ファイルを作成しました: %s\n", configPath)
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
