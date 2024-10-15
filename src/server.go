package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-zoox/chatbot-feishu"
	"github.com/go-zoox/core-utils/regexp"
	"github.com/go-zoox/feishu"
	"github.com/go-zoox/feishu/contact/user"
	"github.com/go-zoox/feishu/event"
	feishuEvent "github.com/go-zoox/feishu/event"
	mc "github.com/go-zoox/feishu/message/content"
	"github.com/go-zoox/logger"
)

// 两种。一种是快速的..一种是长时间执行的

func getUser(request *feishuEvent.EventRequest) (*user.RetrieveResponse, error) {
	sender := request.Sender()
	return &user.RetrieveResponse{
		User: user.UserEntity{
			Name:    sender.SenderID.UserID,
			OpenID:  sender.SenderID.OpenID,
			UnionID: sender.SenderID.UnionID,
			UserID:  sender.SenderID.UserID,
		},
	}, nil
}

func ReplyText(reply func(content string, msgType ...string) error, text string) error {
	if text == "" {
		text = "服务没有返回"
	}

	msgType, content, err := mc.
		NewContent().
		Post(&mc.ContentTypePost{
			ZhCN: &mc.ContentTypePostBody{
				Content: [][]mc.ContentTypePostBodyItem{
					{
						{
							Tag:      "text",
							UnEscape: false,
							Text:     text,
						},
					},
				},
			},
		}).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build content: %v", err)
	}
	if err := reply(string(content), msgType); err != nil {
		logger.Infof("failed to reply: %v", err)
	}

	return nil
}

func getCommand(client feishu.Client, text string, request *feishuEvent.EventRequest) string {
	var command string
	// group chat
	if request.IsGroupChat() {
		botInfo, _ := client.Bot().GetBotInfo()
		if ok := regexp.Match("^@_user_1", text); ok {
			for _, mention := range request.Event.Message.Mentions {
				if mention.Key == "@_user_1" && mention.ID.OpenID == botInfo.OpenID {
					command = strings.Replace(text, "@_user_1", "", 1)
					logger.Infof("chat command %s", command)
					break
				}
			}
		}
	} else if request.IsP2pChat() {
		command = text
	}
	command = strings.TrimSpace(command)
	logger.Infof("chat command %s", command)
	return command
}

func FeishuServer(feishuConf *chatbot.Config) (chatbot.ChatBot, error) {
	bot, err := chatbot.New(feishuConf)
	client := feishu.New(&feishu.Config{"https://open.feishu.cn", feishuConf.AppID, feishuConf.AppSecret})
	if err != nil {
		logger.Errorf("failed to create bot: %v", err)
		return nil, err
	}

	bot.OnCommand("ping", &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := ReplyText(reply, "pong"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}
			return nil
		},
	})

	bot.OnCommand("help", &chatbot.Command{
		Handler: func(args []string, request *event.EventRequest, reply chatbot.MessageReply) error {
			helpText := "直接输入命令，将会发送一个 HTTP 请求到指定的服务。"
			if err := ReplyText(reply, helpText); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}
			return nil
		},
	})

	bot.OnMessage(func(text string, request *event.EventRequest, reply chatbot.MessageReply) error {
		command := getCommand(client, text, request)
		if command == "" {
			logger.Infof("ignore empty command message")
			return nil
		}

		// 忽略以 "/" 开头的命令
		if strings.HasPrefix(command, "/") {
			logger.Infof("ignore command message starting with '/'")
			return nil
		}

		// 发送 HTTP 请求
		err := SendHTTPRequest(command)
		if err != nil {
			ReplyText(reply, fmt.Sprintf("发送 HTTP 请求失败: %v", err))
			return nil
		}

		// ReplyText(reply, fmt.Sprintf("已发送命令: %s", command))
		return nil
	})

	return bot, nil
}

// 修改后的 RunCommand 函数
func SendHTTPRequest(command string) error {
	url := "http://localhost:8000/miio/command"
	payload := map[string]string{
		"command": command,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	return nil
}
