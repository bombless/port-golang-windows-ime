package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {
	candidateDebugLog("gui.start", "args=%q mode=standalone", os.Args[1:])
	projectDir, err := findProjectDir()
	if err != nil {
		projectDir, _ = os.Getwd()
	}

	var mw *walk.MainWindow
	var output *walk.TextEdit
	var status *walk.Label

	appendOutput := func(message string) {
		if mw == nil {
			return
		}
		mw.Synchronize(func() {
			if output != nil {
				output.AppendText(message + "\r\n")
			}
		})
	}

	run := func(label string, args ...string) {
		if status != nil {
			status.SetText(label + "：运行中…")
		}
		appendOutput("\r\n> go " + strings.Join(args, " "))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", args...)
			cmd.Dir = projectDir
			cmd.Env = os.Environ()
			if strings.Contains(strings.Join(args, " "), "-buildmode=c-shared") {
				cmd.Env = append(cmd.Env, "CGO_LDFLAGS=-lole32 -luuid -loleaut32")
			}
			result, commandErr := cmd.CombinedOutput()
			message := strings.TrimSpace(string(result))
			if message == "" {
				message = "（命令没有输出）"
			}
			appendOutput(message)
			mw.Synchronize(func() {
				if commandErr != nil {
					status.SetText(label + "：失败 - " + commandErr.Error())
				} else {
					status.SetText(label + "：完成")
				}
			})
		}()
	}

	err = (MainWindow{
		AssignTo: &mw,
		Title:    "Go Windows IME",
		MinSize:  Size{Width: 760, Height: 480},
		Size:     Size{Width: 900, Height: 600},
		Layout:   VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}},
		Children: []Widget{
			Label{Text: "独立的 Walk 桌面控制台：运行测试、构建输入法 TSF DLL，并查看诊断日志。此窗口不会自动注册或安装输入法。"},
			Label{Text: "项目目录：" + projectDir},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "运行 Go 测试", OnClicked: func() { run("测试", "test", "./...") }},
					PushButton{Text: "构建 TSF DLL", OnClicked: func() {
						if runtime.GOOS != "windows" {
							status.SetText("构建 TSF DLL 需要在 Windows 上执行")
							return
						}
						if err := os.MkdirAll(filepath.Join(projectDir, "build"), 0755); err != nil {
							status.SetText("创建 build 目录失败：" + err.Error())
							return
						}
						run("构建 DLL", "build", "-buildmode=c-shared", "-o", filepath.Join("build", "PortGoWindowsIME.dll"), "./tip")
					}},
					PushButton{Text: "打开项目目录", OnClicked: func() {
						if err := exec.Command("explorer.exe", projectDir).Start(); err != nil {
							status.SetText("打开目录失败：" + err.Error())
						}
					}},
					PushButton{Text: "打开运行日志", OnClicked: func() {
						logPath := filepath.Join(os.TempDir(), "TypeScriptWindowsIME.log")
						if _, err := os.Stat(logPath); err != nil {
							status.SetText("日志文件尚未生成：" + logPath)
							return
						}
						if err := exec.Command("notepad.exe", logPath).Start(); err != nil {
							status.SetText("打开日志失败：" + err.Error())
						}
					}},
				},
			},
			Label{AssignTo: &status, Text: "就绪"},
			TextEdit{AssignTo: &output, ReadOnly: true, VScroll: true, HScroll: true},
		},
	}).Create()
	if err != nil {
		fmt.Fprintln(os.Stderr, "创建窗口失败：", err)
		return
	}
	mw.Run()
}

func findProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Prefer the repository root, not this GUI's nested module.
			if _, err := os.Stat(filepath.Join(dir, "tip")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("找不到项目根目录（需要包含 go.mod 和 tip 目录）")
}
