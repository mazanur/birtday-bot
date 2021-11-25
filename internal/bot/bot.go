package bot

import (
	"context"
	"github.com/almaznur91/splitty/internal/api"
	"github.com/go-pkgz/syncs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const start string = "/start"

//actions
const (
	setBirtDate api.Action = "set_birt_date"
	createRoom  api.Action = "create_room"
	joinRoom    api.Action = "join_room"
	viewStart   api.Action = "start"
	viewRoom    api.Action = "viewRoom"

	chooseOperations   api.Action = "choose_operations"
	chooseDebts        api.Action = "choose_debts"
	roomSetting        api.Action = "room_setting"
	viewAllRooms       api.Action = "user_setting"
	statistics         api.Action = "archive_room"
	wantDonorOperation api.Action = "archive_room"
)

// Interface is a bot reactive spec. response will be sent if "send" result is true
type Interface interface {
	OnMessage(ctx context.Context, update *api.Update) (api.TelegramMessage, error)
	HasReact(update *api.Update) bool
}

// SuperUser defines interface checking ig user name in su list
type SuperUser interface {
	IsSuper(userName string) bool
}

// MultiBot combines many bots to one virtual
type MultiBot []Interface

// OnMessage pass msg to all bots and collects reposnses (combining all of them)
//noinspection GoShadowedVar
func (b MultiBot) OnMessage(ctx context.Context, update *api.Update) (api.TelegramMessage, error) {

	resps := make(chan api.TelegramMessage)
	errors := make(chan error)

	wg := syncs.NewSizedGroup(4)
	for _, bot := range b {
		bot := bot
		wg.Go(func(ctx context.Context) {
			if bot.HasReact(update) {
				resp, err := bot.OnMessage(ctx, update)
				if err != nil {
					errors <- err
				} else {
					resps <- resp
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(resps)
		close(errors)
	}()

	message := &api.TelegramMessage{Chattable: []tgbotapi.Chattable{}}
	var eror error

tobreake:
	for {
		select {
		case r, ok := <-resps:
			if !ok {
				break tobreake
			}
			message.Chattable = append(message.Chattable, r.Chattable...)
			message.InlineConfig = r.InlineConfig
			message.CallbackConfig = r.CallbackConfig
			message.Redirect = r.Redirect
			message.Send = true

		case err, ok := <-errors:
			if !ok {
				break tobreake
			}
			eror = err

		default:
		}
	}

	return *message, eror
}

func (b MultiBot) HasReact(u *api.Update) bool {
	var hasReact bool
	for _, bot := range b {
		hasReact = hasReact && bot.HasReact(u)
	}
	return hasReact
}
