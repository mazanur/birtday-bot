package service

import (
	"context"
	"github.com/almaznur91/splitty/internal/api"
	"github.com/almaznur91/splitty/internal/repository"
)

type RoomService struct {
	repository.RoomRepository
}

func NewRoomService(r repository.RoomRepository) *RoomService {
	return &RoomService{r}
}

func (rs *RoomService) CreateRoom(ctx context.Context, r *api.Room) (*api.Room, error) {
	rId, err := rs.RoomRepository.SaveRoom(ctx, r)
	r.ID = rId
	return r, err
}
