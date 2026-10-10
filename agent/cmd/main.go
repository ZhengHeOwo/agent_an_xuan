package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/ZhengHeOwo/agent_noah/agent/internal/agent"
	"github.com/ZhengHeOwo/agent_noah/agent/internal/config"
	"github.com/ZhengHeOwo/agent_noah/agent/internal/model/openai"
	"github.com/ZhengHeOwo/agent_noah/agent/internal/terminal"
	"github.com/ZhengHeOwo/agent_noah/agent/internal/tool"
	"github.com/ZhengHeOwo/agent_noah/agent/internal/websearch"
	"github.com/ZhengHeOwo/agent_noah/agent/internal/workspace"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	console, err := terminal.NewConsole(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("创建终端交互对象失败: %w", err)
	}
	defer console.Close()

	if err := config.LoadEnvFile("./agent/local/config/.env.local"); err != nil {
		return fmt.Errorf("加载环境文件失败: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载程序配置失败: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.Model.Timeout,
	}

	client, err := openai.NewClient(httpClient, cfg.Model.Endpoint, cfg.Model.APIKey)
	if err != nil {
		return fmt.Errorf("客户端配置失败: %w", err)
	}

	if err := os.MkdirAll(workspace.DefaultDir, 0o700); err != nil {
		return fmt.Errorf("create workspace directory %q: %w", workspace.DefaultDir, err)
	}

	projectWorkspace, err := workspace.OpenWorkspace(workspace.DefaultDir)
	if err != nil {
		return fmt.Errorf("open workspace %q: %w", workspace.DefaultDir, err)
	}
	defer func() {
		_ = projectWorkspace.Close()
	}()

	readTextFileTool, err := workspace.NewReadTextFileTool(projectWorkspace)
	if err != nil {
		return fmt.Errorf("create read_text_file tool: %w", err)
	}

	listTextFilesTool, err := workspace.NewListTextFilesTool(projectWorkspace)
	if err != nil {
		return fmt.Errorf("create list_text_files tool: %w", err)
	}

	writeTextFileTool, err := workspace.NewWriteTextFileTool(projectWorkspace, console)
	if err != nil {
		return fmt.Errorf("create write_text_file tool: %w", err)
	}

	searchTextTool, err := workspace.NewSearchTextTool(projectWorkspace)
	if err != nil {
		return fmt.Errorf("create search_text tool: %w", err)
	}

	webSearchTool, err := websearch.NewTool(cfg.WebSearch.BraveAPIKey)
	if err != nil {
		return fmt.Errorf("create web_search tool: %w", err)
	}

	toolsRegistry, err := tool.NewToolRegistry(
		readTextFileTool,
		listTextFilesTool,
		writeTextFileTool,
		searchTextTool,
		webSearchTool,
	)

	if err != nil {
		return fmt.Errorf("工具注册表创建失败: %w", err)
	}

	runtime, err := agent.NewRuntime(
		client,
		cfg.Model.Name,
		cfg.Agent.SystemPrompt,
		toolsRegistry,
	)
	if err != nil {
		return fmt.Errorf("创建Agent运行器失败: %w", err)
	}

console.Write([]byte(fmt.Sprint("Agent Noah 已启动, 输入 exit 退出\n")))

	for {
		input, err := console.ReadLine(": ")
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("读取终端失败: %w", err)
		}

		if input == "" {
			continue
		}

		if input == "exit" {
			return nil
		}

		reply, err := runtime.RunTurn(context.Background(), input)
		if err != nil {
			console.Write([]byte(fmt.Sprintf("本轮执行失败: %v\n", err)))
			continue
		}

		console.Write([]byte(fmt.Sprint("\nNoah: \n%s\n", reply)))
	}
}
