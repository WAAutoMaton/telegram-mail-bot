# telegram-mail-bot

## Setup

Compile:
```shell
git clone https://github.com/WAAutoMaton/telegram-mail-bot.git
cd telegram-mail-bot
go mod tidy
go build ./
```

Configure:
```shell
mkdir config
touch config/config.json
```

Edit the ``config.json``, for example:
```json
{
  "token": "<Your bot token> : string",
  "imapserver": "imap.example.com:993",
  "smtpserver": "smtp.example.com:589",
  "smtphost": "smtp.example.com",
  "username": "user@example.com",
  "password": "Your Mail's Password",
  "uid": "<Your telegram digital ID> : int"
}
```

(Optional) If you want to use proxy:
```shell
export http_proxy="Your proxy"
```

Finally

```shell
./telegram-mail-bot
```

# Usage

Use ``/help`` to get more information.
