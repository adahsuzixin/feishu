FROM golang:1.18 as builder

ENV GOPROXY https://goproxy.cn

# mod
WORKDIR /app
COPY . /app/
RUN go mod download
RUN go build -o dist/feishu_shell_bot ./main.go

FROM golang:1.18
USER root
COPY --from=builder /app/dist/feishu_shell_bot /app/feishu_shell_bot
RUN chmod +x /app/feishu_shell_bot
ENV FEISHU_APP_ID=cli_a54ebbe665b9d00c
ENV FEISHU_APP_SECRET=9iLN3wzMdTa6vXPLUTzYNgxsJXZ70EI8
ENV FEISHU_ENCRYPT_KEY=TBG4oYF5iJnStSucOcoGmcXUGg8o1aRM
ENV FEISHU_VERIFICATION_TOKEN=TBG4oYF5iJnStSucOcoGmcXUGg8o1aRM
ENV FEISHU_BOT_PATH=/
ENV FEISHU_BOT_PORT=8081
WORKDIR /app/

ENTRYPOINT [ "/app/feishu_shell_bot" ]
