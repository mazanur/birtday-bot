package service

import (
	"context"
	"github.com/almaznur91/splitty/internal/api"
	"github.com/almaznur91/splitty/internal/repository"
	"github.com/rs/zerolog/log"
)

func NewUserService(r repository.UserRepository) *UserService {
	return &UserService{r}
}
func NewChatStateService(r repository.ChatStateRepository) *ChatStateService {
	return &ChatStateService{r}
}

func NewButtonService(r repository.ButtonRepository) *ButtonService {
	return &ButtonService{r}
}

type UserService struct {
	repository.UserRepository
}

type ChatStateService struct {
	repository.ChatStateRepository
}

type ButtonService struct {
	repository.ButtonRepository
}

func (css *ChatStateService) CleanChatState(ctx context.Context, state *api.ChatState) {
	if state == nil {
		return
	} else if err := (*css).DeleteByUserId(ctx, state.UserId); err != nil {
		log.Error().Err(err).Msg("")
	}
}

func containsUserId(users *[]api.User, id int64) bool {
	for _, u := range *users {
		if u.ID == id {
			return true
		}
	}
	return false
}

func deleteUser(users []api.User, userId int64) []api.User {
	index := -1
	for i, v := range users {
		if v.ID == userId {
			index = i
			break
		}
	}
	if index == -1 {
		return users
	}
	copy(users[index:], users[index+1:])
	return users[:len(users)-1]
}
