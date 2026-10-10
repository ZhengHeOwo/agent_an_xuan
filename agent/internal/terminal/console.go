package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ZhengHeOwo/agent_noah/agent/internal/tool"

	"github.com/chzyer/readline"
)

// Console 统一管理命令行程序的文本输入和输出。
//
// 同一个Console应同时用于主对话和副作用确认，避免多个缓冲读取器
// 竞争同一个输入源。
type Console struct {
	rl *readline.Instance
}

// NewConsole 创建终端交互对象。
func NewConsole(reader io.Reader, writer io.Writer) (*Console, error) {
	if reader == nil {
		return nil, fmt.Errorf("终端输入不能为空")
	}

	if writer == nil {
		return nil, fmt.Errorf("终端输出不能为空")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 ": ",
		DisableAutoSaveHistory: true,
		HistoryLimit:           100,
		Stdin:                  io.NopCloser(reader),
		Stdout:                 writer,
		Stderr:                 writer,
	})
	if err != nil {
		return nil, fmt.Errorf("readline 初始化失败 %w", err)
	}

	return &Console{rl: rl}, nil
}

// ReadLine 输出提示并读取一行文本。
func (c *Console) ReadLine(prompt string) (string, error) {
	c.rl.SetPrompt(prompt)
	line, err := c.rl.Readline()
	if err != nil {
		if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
			return "", io.EOF
		}

		return "", fmt.Errorf("读取终端输入失败: %w", err)
	}

	return strings.TrimSpace(line), nil
}

// Confirm 展示副作用操作并等待明确授权。
func (c *Console) Confirm(ctx context.Context, request tool.ConfirmationRequest) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("确认操作前上下文已结束: %w", err)
	}

	if strings.TrimSpace(request.Action) == "" {
		return false, fmt.Errorf("确认操作标识不能为空")
	}

	if strings.TrimSpace(request.Summary) == "" {
		return false, fmt.Errorf("确认操作摘要不能为空")
	}

	prompt := fmt.Sprintf(
		"\n需要确认操作\n\n操作: %s\n\n摘要: %s\n\n详情: \n%s\n\n确认执行?请输入 y 或 n: ",
		request.Action,
		request.Summary,
		request.Details,
	)

	for {
		decision, err := c.ReadLine(prompt)
		if err != nil {
			return false, fmt.Errorf("读取确认结果失败: %w", err)
		}

		switch strings.ToLower(decision) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			prompt = "输入无效, 请输入 y 或 n: "
		}
	}
}

var _ tool.Confirmer = (*Console)(nil)

func (c *Console) Write(p []byte) (int, error) {
	return c.rl.Write(p)
}

func (c *Console) Close() error {
	return c.rl.Close()
}
