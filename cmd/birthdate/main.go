package main

import (
	"context"
	"fmt"
	"github.com/almaznur91/splitty/internal/handler"
	"github.com/gookit/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/language"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/almaznur91/splitty/internal/bot"
	"github.com/almaznur91/splitty/internal/events"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xlab/closer"
)

var revision = "local"

func main() {
	defer closer.Close()
	ctx := context.Background()

	cfg, err := initConfig()
	if err != nil {
		log.Err(err).Msg("Can not init config")
	}

	initI18n(cfg)

	if err := initLogger(cfg); err != nil {
		log.Fatal().Err(err).Msg("Can not init logger")
	}

	rand.Seed(int64(time.Now().Nanosecond()))

	app, cl, err := initApp(ctx, cfg)
	if err != nil {
		log.Error().Err(err).Msg("Can not init application")
		return
	}
	closer.Bind(cl)

	if err := app.Do(ctx); err != nil {
		log.Error().Err(err).Msg("telegram listener failed")
		return
	}
}

type tgLogger struct {
	zerolog.Logger
}

func (i *tgLogger) Println(v ...interface{}) { i.Print(v...) }

func initTelegramApi(cfg *config, bcfg *bot.Config) (*tbapi.BotAPI, error) {
	_ = tbapi.SetLogger(&tgLogger{log.Output(zerolog.ConsoleWriter{Out: os.Stdout})})
	tbAPI, err := tbapi.NewBotAPI(cfg.TgToken)
	if err != nil {
		log.Error().Err(err).Msg("can't make telegram bot")
		return nil, err
	}
	log.Info().Msg("super users: " + strings.Join(cfg.SuperUsers, ","))

	bcfg.BotName = tbAPI.Self.UserName
	tbAPI.Debug = cfg.TgDebug
	log.Info().Msgf("BotName: %s", bcfg.BotName)
	return tbAPI, nil
}

func initTelegramConfig(tbAPI *tbapi.BotAPI, bots []bot.Interface, bs events.ButtonService, us events.UserService, cs events.ChatStateService, eh *handler.ErrorHandler) (*events.TelegramListener, error) {
	multiBot := bot.MultiBot(bots)

	tgListener := &events.TelegramListener{
		TbAPI:            tbAPI,
		Bots:             multiBot,
		ErrorHandler:     eh,
		ChatStateService: cs,
		ButtonService:    bs,
		UserService:      us,
	}

	return tgListener, nil
}

func initLogger(c *config) error {
	log.Debug().Msg("initialize logger")
	logLvl, err := zerolog.ParseLevel(strings.ToLower(c.LogLevel))
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(logLvl)
	switch c.LogFmt {
	case "console":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	case "json":
	default:
		return fmt.Errorf("unknown output format %service", c.LogFmt)
	}
	return nil
}

func initMongoConnection(ctx context.Context, cfg *config) (*mongo.Database, func(), error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(cfg.DbAddr))
	if err != nil {
		return nil, nil, err
	}

	// Create connect
	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	return client.Database(cfg.DbName), func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal().Err(err).Msg("error while connect to mongo")
		}
	}, nil
}

func initBotConfig(c *config) *bot.Config {
	cfg := &bot.Config{
		SuperUsers: c.SuperUsers,
	}
	return cfg
}

func initI18n(c *config) {
	languages := map[string]string{
		language.English.String(): "English",
		language.Russian.String(): "??????????????",
	}
	i18n.Init("conf/lang", c.DefaultLanguage, languages)
}
