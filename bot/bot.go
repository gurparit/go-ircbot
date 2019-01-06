package bot

import (
	"encoding/json"
	"fmt"
	"github.com/gurparit/go-common/env"
	"github.com/gurparit/go-ircbot/conf"
	"strings"

	"github.com/gurparit/go-ircbot/command"
	irc "github.com/gurparit/go-ircevent"
)

// Bot irc bot object
type Bot struct {
	keys   *conf.Keys
	config *Config
	conn   *irc.Connection
}

// Config irc bot config
type Config struct {
	Server   string
	Username string
	Password string
	UseTLS   bool
	Debug    bool

	Channels []string

	ExtendedCommands bool

	MessageListeners chan string
}

var functions = make(map[string]command.Command)

func defaultCommands() {
	addCommand("echo", command.Echo{})
	addCommand("time", command.Time{})
	addCommand("go", command.Hello{})
	addCommand("so", command.Shoutout{})
}

func extendedCommands() {
	addCommand("g", command.Google{})
	addCommand("ud", command.Urban{})
	addCommand("yt", command.Youtube{})
	addCommand("gif", command.Giphy{})
	addCommand("define", command.Oxford{})
	addCommand("ety", command.Oxford{Etymology: true})
}

func addCommand(key string, cmd command.Command) {
	functions["!"+key] = cmd
}

func recovery() {
	if r := recover(); r != nil {
		fmt.Println(r)
	}
}

func (*Bot) onNewMessage(response command.Response, event command.MessageEvent) {
	defer recovery()

	params := strings.SplitN(event.Message, " ", 2)
	action := params[0]
	query := ""

	if len(params) > 1 {
		query = params[1]
	}

	event.Message = query

	if c, ok := functions[action]; ok {
		c.Execute(response, event)
	}
}

func (bot *Bot) onWelcomeEvent(channels []string) func(*irc.Event) {
	return func(event *irc.Event) {
		for _, channel := range channels {
			bot.conn.Join(channel)
		}
	}
}

func (bot *Bot) onMessageEvent(event *irc.Event) {
	channel := event.Arguments[0]
	user := event.Nick
	message := event.Message()
	tags := event.Tags

	messageEvent := command.MessageEvent{
		Channel:  channel,
		Username: user,
		Message:  message,
		Tags:     tags,
	}

	events := bot.config.MessageListeners
	if events != nil {
		stringData, _ := json.Marshal(messageEvent)
		events <- fmt.Sprintf(string(stringData))
	}

	if strings.HasPrefix(message, "!") {
		go bot.onNewMessage(bot.getResponseHandlerForChannel(channel), messageEvent)
	}
}

func (bot *Bot) getResponseHandlerForChannel(channel string) command.Response {
	return func(response string) {
		bot.conn.Privmsg(channel, response)
	}
}

// New returns a new IRCBot with supplied config
func New(cfg *Config) *Bot {
	return &Bot{config: cfg}
}

// Default returns a new IRCBot with default config
func Default(server, username, password string) *Bot {
	return New(&Config{
		Server:   server,
		Username: username,
		Password: password,
		UseTLS:   false,
		Debug:    true,
		Channels: []string{"#general"},

		ExtendedCommands: false,
	})
}

// DefaultTLS returns a new IRCBot with default config and TLS enabled
func DefaultTLS(server, username, password string) *Bot {
	return New(&Config{
		Server:   server,
		Username: username,
		Password: password,
		UseTLS:   true,
		Debug:    true,
		Channels: []string{"#general"},

		ExtendedCommands: false,
	})
}

const (
	// EventWelcome callback key for welcome event
	EventWelcome = "001"
	// EventPrivateMessage callback key for private message event
	EventPrivateMessage = "PRIVMSG"
	// EventCap callback for cap event
	EventCap = "CAP"
)

func (bot *Bot) negotiateCaps() {
	// 2019/01/06 19:01:22 --> CAP LS
	// 2019/01/06 19:01:22 <-- :tmi.twitch.tv CAP * LS :twitch.tv/tags twitch.tv/commands twitch.tv/membership
	// 2019/01/06 19:01:22 --> CAP REQ :twitch.tv/tags
	// 2019/01/06 19:01:23 <-- :tmi.twitch.tv CAP * ACK :twitch.tv/tags
	// 2019/01/06 19:01:23 --> CAP END
	bot.conn.SendRaw("CAP LS")
}

func (bot *Bot) onCapEvent(event *irc.Event) {
	if event.Arguments[1] == "LS" {
		bot.conn.SendRaw("CAP REQ :twitch.tv/tags")
	}

	if event.Arguments[1] == "ACK" {
		bot.conn.SendRaw("CAP END")
	}
}

// Start bot start
func (bot *Bot) Start() {
	username := bot.config.Username

	conn := irc.IRC(username, username)
	conn.UseTLS = bot.config.UseTLS
	conn.Debug = bot.config.Debug
	conn.Password = bot.config.Password

	bot.conn = conn

	conn.AddCallback(EventWelcome, bot.onWelcomeEvent(bot.config.Channels))
	conn.AddCallback(EventPrivateMessage, bot.onMessageEvent)
	conn.AddCallback(EventCap, bot.onCapEvent)

	defaultCommands()
	if bot.config.ExtendedCommands {
		keys := conf.Keys{}
		env.Read(&keys)

		command.KeyValues = &keys
		extendedCommands()
	}

	conn.Connect(bot.config.Server)

	bot.negotiateCaps()
	conn.Loop()
}
