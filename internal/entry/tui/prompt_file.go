package tui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/ainovel-cli/internal/entry/startup"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/utils"
)

func startPromptFile(m Model, args []string) (tea.Model, tea.Cmd) {
	if m.mode != modeNew {
		m.applyEvent(host.Event{
			Time:     time.Now(),
			Category: "ERROR",
			Summary:  "/prompt-file chỉ dùng được ở màn hình tạo truyện mới",
			Level:    "error",
		})
		m.refreshEventViewport()
		return m, nil
	}
	if len(args) != 1 {
		m.applyEvent(host.Event{
			Time:     time.Now(),
			Category: "ERROR",
			Summary:  "Cách dùng: /prompt-file <đường_dẫn_file>",
			Level:    "error",
		})
		m.refreshEventViewport()
		return m, nil
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		m.applyEvent(host.Event{
			Time:     time.Now(),
			Category: "ERROR",
			Summary:  "Đọc prompt file thất bại: " + err.Error(),
			Level:    "error",
		})
		m.refreshEventViewport()
		return m, nil
	}
	prompt := utils.CollapseBlankLines(string(data))
	if prompt == "" {
		m.applyEvent(host.Event{
			Time:     time.Now(),
			Category: "ERROR",
			Summary:  "Prompt file rỗng: " + args[0],
			Level:    "error",
		})
		m.refreshEventViewport()
		return m, nil
	}

	m.err = nil
	m.textarea.Reset()
	m.refitTextareaHeight()
	if m.startupMode == startupModeQuick {
		plan, err := startup.PrepareQuick(startup.Request{
			Mode:        startup.ModeQuick,
			UserPrompt:  prompt,
			OutputDir:   m.runtime.Dir(),
			Interactive: true,
		})
		if err != nil {
			m.err = fmt.Errorf("chuẩn bị prompt file thất bại: %w", err)
			return m, nil
		}
		return m, startRuntime(m.runtime, plan)
	}

	m.cocreate = newCoCreateState(prompt)
	return m, m.sendCoCreate()
}
