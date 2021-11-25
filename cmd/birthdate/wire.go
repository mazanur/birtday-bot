//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"github.com/almaznur91/splitty/internal/bot"
	"github.com/almaznur91/splitty/internal/events"
	"github.com/almaznur91/splitty/internal/handler"
	"github.com/almaznur91/splitty/internal/repository"
	"github.com/almaznur91/splitty/internal/service"
	"github.com/google/wire"
)

func initApp(ctx context.Context, cfg *config) (tg *events.TelegramListener, closer func(), err error) {
	wire.Build(initMongoConnection, initTelegramApi, initTelegramConfig, initBotConfig, handler.NewErrorHandler,
		service.NewUserService, wire.Bind(new(bot.UserService), new(*service.UserService)),
		wire.Bind(new(events.UserService), new(*service.UserService)),
		service.NewChatStateService, wire.Bind(new(bot.ChatStateService), new(*service.ChatStateService)),
		service.NewButtonService, wire.Bind(new(bot.ButtonService), new(*service.ButtonService)),
		service.NewRoomService, wire.Bind(new(bot.RoomService), new(*service.RoomService)),
		wire.Bind(new(events.ChatStateService), new(*service.ChatStateService)),
		wire.Bind(new(events.ButtonService), new(*service.ButtonService)),
		ProvideBotList, bots,
		repository.NewUserRepository, wire.Bind(new(repository.UserRepository), new(*repository.MongoUserRepository)),
		repository.NewChatStateRepository, wire.Bind(new(repository.ChatStateRepository), new(*repository.MongoChatStateRepository)),
		repository.NewRoomRepository, wire.Bind(new(repository.RoomRepository), new(*repository.MongoRoomRepository)),
		repository.NewButtonRepository, wire.Bind(new(repository.ButtonRepository), new(*repository.MongoButtonRepository)),
	)
	return nil, nil, nil
}

var bots = wire.NewSet(bot.NewStartScreen, bot.NewRoomSetName, bot.NewRoomCreating, bot.NewStartScreenInitPerson)

func ProvideBotList(b2 *bot.StartScreen, b3 *bot.RoomCreating, b4 *bot.RoomSetName, b5 *bot.StartScreenInitPerson) []bot.Interface {
	return []bot.Interface{b2, b3, b4}
}
