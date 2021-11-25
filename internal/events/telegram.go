package events

import (
	"context"
	"github.com/almaznur91/splitty/internal/api"
	"github.com/almaznur91/splitty/internal/bot"
	"github.com/almaznur91/splitty/internal/handler"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ChatStateService interface {
	FindByUserId(ctx context.Context, userId int64) (*api.ChatState, error)
}

type ButtonService interface {
	FindById(ctx context.Context, id string) (*api.Button, error)
}

type UserService interface {
	UpsertUser(ctx context.Context, u api.User) (*api.User, error)
}

// TelegramListener listens to tg update, forward to bots and send back responses
// Not thread safe
type TelegramListener struct {
	TbAPI            tbAPI
	Bots             bot.Interface
	ErrorHandler     *handler.ErrorHandler
	ChatStateService ChatStateService
	ButtonService    ButtonService
	upds             chan tbapi.Update
	UserService      UserService
}

type tbAPI interface {
	GetUpdatesChan(config tbapi.UpdateConfig) tbapi.UpdatesChannel
	Send(c tbapi.Chattable) (tbapi.Message, error)
	GetChat(config tbapi.ChatInfoConfig) (tbapi.Chat, error)
}

// Do process all events, blocked call
func (l *TelegramListener) Do(ctx context.Context) (err error) {

	go l.ErrorHandler.Do(ctx)

	u := tbapi.NewUpdate(0)
	u.Timeout = 60

	var updates tbapi.UpdatesChannel
	updates = l.TbAPI.GetUpdatesChan(u)

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case update, ok := <-updates:

			if !ok {
				return errors.Errorf("telegram update chan closed")
			}

			upd := transformUpdate(update)

			user, err := getFrom(upd)
			if err != nil {
				l.ErrorHandler.HandleErrorWithMsg(err, "failed define user")
				break
			}

			upd.User, err = l.UserService.UpsertUser(ctx, *user)
			if err != nil {
				l.ErrorHandler.HandleErrorWithMsg(err, "failed to upsert user")
			}

			if err := l.populateBtn(ctx, upd); err != nil {
				l.ErrorHandler.HandleErrorWithMsg(err, "failed to populateBtn")
			}

			if err := l.populateChatState(ctx, upd); err != nil {
				l.ErrorHandler.HandleErrorWithMsg(err, "failed to populateChatState")
			}

			log.Debug().Msgf("incoming msg: %+v; btn:%+v", upd.Message, upd.Button)

			l.processUpdate(ctx, upd)
		}
	}
}

func (l *TelegramListener) processUpdate(ctx context.Context, upd *api.Update) {
	resp, err := l.Bots.OnMessage(ctx, upd)
	if err != nil {
		l.ErrorHandler.HandleError(err)
	}

	if err := l.sendBotResponse(ctx, resp); err != nil {
		l.ErrorHandler.HandleError(err)
	}
}

func (l *TelegramListener) populateBtn(ctx context.Context, upd *api.Update) error {
	if upd.CallbackQuery != nil {
		btn, err := l.ButtonService.FindById(ctx, upd.CallbackQuery.Data)
		if err != nil {
			return errors.Wrapf(err, "failed to find Button by id %q", err)
		}
		upd.Button = btn
	}
	return nil
}

func (l *TelegramListener) populateChatState(ctx context.Context, upd *api.Update) error {
	var userId int64
	if upd.Message != nil {
		userId = upd.Message.From.ID
	} else if upd.CallbackQuery != nil && upd.CallbackQuery.Message != nil {
		userId = upd.CallbackQuery.From.ID
	}

	cs, err := l.ChatStateService.FindByUserId(ctx, userId)
	if err != nil {
		return errors.Wrapf(err, "failed to find ChatState by id %q", err)
	}
	upd.ChatState = cs
	return nil
}

// sendBotResponse sends bot'service answer to tg channel and saves it to log
func (l *TelegramListener) sendBotResponse(ctx context.Context, resp api.TelegramMessage) error {
	if !resp.Send {
		return nil
	}

	if resp.InlineConfig != nil {
		response, err := l.TbAPI.Send(*resp.InlineConfig)
		if err != nil {
			return errors.Wrapf(err, "can't send query to telegram %v", response)
		}
		log.Debug().Msgf("bot response - %q", resp.InlineConfig)
	}

	if len(resp.Chattable) > 0 {
		for _, v := range resp.Chattable {
			if v == nil {
				continue
			}
			response, err := l.TbAPI.Send(v)
			if err != nil {
				return errors.Wrapf(err, "can't send message to telegram %v", v)
			}
			log.Debug().Msgf("bot response chat - %v, text - %v, messageId - %v", response.Chat, response.Text, response.MessageID)
		}
	}
	if resp.CallbackConfig != nil {
		response, err := l.TbAPI.Send(*resp.CallbackConfig)
		if err != nil {
			return errors.Wrapf(err, "can't send calback to telegram %v", resp.CallbackConfig)
		}
		log.Debug().Msgf("bot response - %+v", response)
	}
	if resp.Redirect != nil {
		if resp.Redirect.FromRedirect {
			log.Error().Stack().Msg("recursive multiple redirection")
		} else {
			resp.Redirect.FromRedirect = true
			l.processUpdate(ctx, resp.Redirect)
		}
	}
	return nil
}

func transform(msg *tbapi.Message) *api.Message {
	if msg == nil {
		return nil
	}
	message := api.Message{
		ID:   msg.MessageID,
		Sent: msg.Time(),
		Text: msg.Text,
	}

	message.Chat = &api.Chat{
		ID:   msg.Chat.ID,
		Type: msg.Chat.Type,
	}

	if msg.From != nil {
		message.From = transformUser(msg.From)
	}

	switch {
	case msg.Entities != nil && len(msg.Entities) > 0:
		message.Entities = transformEntities(msg.Entities)

	case msg.Document != nil:
		message.Document = &api.Document{
			FileID:   msg.Document.FileID,
			FileSize: msg.Document.FileSize,
			MimeType: msg.Document.MimeType,
		}

	case msg.Video != nil:
		message.Video = &api.Video{
			FileID:   msg.Video.FileID,
			FileSize: msg.Video.FileSize,
			MimeType: msg.Video.MimeType,
		}

	case msg.Photo != nil && len(msg.Photo) > 0:
		sizes := msg.Photo
		lastSize := sizes[len(sizes)-1]
		message.Image = &api.Image{
			FileID:   lastSize.FileID,
			Width:    lastSize.Width,
			Height:   lastSize.Height,
			Caption:  msg.Caption,
			Entities: transformEntities(msg.CaptionEntities),
		}
	}

	return &message
}

func transformUpdate(u tbapi.Update) *api.Update {
	update := &api.Update{}

	if u.CallbackQuery != nil {
		update.CallbackQuery = &api.CallbackQuery{
			ID:              u.CallbackQuery.ID,
			From:            transformUser(u.CallbackQuery.From),
			Message:         transform(u.CallbackQuery.Message),
			InlineMessageID: u.CallbackQuery.InlineMessageID,
			Data:            u.CallbackQuery.Data,
		}
	}

	if u.InlineQuery != nil {
		i := u.InlineQuery
		update.InlineQuery = &api.InlineQuery{
			ID:     i.ID,
			Query:  i.Query,
			Offset: i.Offset,
			From:   transformUser(i.From),
		}
	}

	if u.EditedMessage != nil {
		update.Message = transform(u.EditedMessage)
	}

	if u.Message != nil {
		update.Message = transform(u.Message)
	}
	return update
}

func transformUser(i *tbapi.User) api.User {
	return api.User{
		ID:          i.ID,
		Username:    i.UserName,
		DisplayName: i.FirstName + " " + i.LastName,
		UserLang:    i.LanguageCode,
	}
}

func transformEntities(entities []tbapi.MessageEntity) *[]api.Entity {
	if entities == nil || len(entities) == 0 {
		return nil
	}

	var result []api.Entity
	for _, entity := range entities {
		e := api.Entity{
			Type:   entity.Type,
			Offset: entity.Offset,
			Length: entity.Length,
			URL:    entity.URL,
		}
		if entity.User != nil {
			e.User = &api.User{
				ID:          entity.User.ID,
				Username:    entity.User.UserName,
				DisplayName: entity.User.FirstName + " " + entity.User.LastName,
			}
		}
		result = append(result, e)
	}

	return &result
}

func getFrom(update *api.Update) (*api.User, error) {
	var user api.User
	if update.CallbackQuery != nil {
		user = update.CallbackQuery.From
	} else if update.Message != nil {
		user = update.Message.From
	} else if update.InlineQuery != nil {
		user = update.InlineQuery.From
	} else {
		return nil, errors.Errorf("Not define user, update - %v", update)
	}
	return &user, nil
}
