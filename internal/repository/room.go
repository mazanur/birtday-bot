package repository

import (
	"context"
	"errors"
	"github.com/almaznur91/splitty/internal/api"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const descParameter = -1
const ascParameter = 1

type MongoRoomRepository struct {
	col *mongo.Collection
}

func NewRoomRepository(col *mongo.Database) *MongoRoomRepository {
	return &MongoRoomRepository{col: col.Collection("room")}
}

type RoomRepository interface {
	FindById(ctx context.Context, id string) (*api.Room, error)
	JoinToRoom(ctx context.Context, u api.User, roomId string) error
	LeaveRoom(ctx context.Context, userId int64, roomId string) error
	SaveRoom(ctx context.Context, r *api.Room) (primitive.ObjectID, error)
	FindRoomsByUserId(ctx context.Context, id int64) (*[]api.Room, error)
	FindArchivedRoomsByUserId(ctx context.Context, id int64) (*[]api.Room, error)
	FindRoomsByLikeName(ctx context.Context, userId int64, name string) (*[]api.Room, error)
	ArchiveRoom(ctx context.Context, userId int64, roomId string) error
	UnArchiveRoom(ctx context.Context, userId int64, roomId string) error
}

func (rr MongoRoomRepository) FindById(ctx context.Context, id string) (*api.Room, error) {
	hex, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	res := rr.col.FindOne(ctx, bson.D{{"_id", bson.D{{"$eq", hex}}}})
	if res.Err() != nil {
		return nil, res.Err()
	}
	rm := &api.Room{}
	if err := res.Decode(rm); err != nil {
		return nil, err
	}
	return rm, nil
}

func (rr MongoRoomRepository) JoinToRoom(ctx context.Context, u api.User, roomId string) error {
	hex, err := primitive.ObjectIDFromHex(roomId)
	if err != nil {
		return err
	}
	hasUserInRoom, err := rr.hasUserInRoom(ctx, u.ID, hex)
	if err != nil || hasUserInRoom {
		return err
	}

	filter := bson.D{{"_id", bson.D{{"$eq", hex}}}}
	_, err = rr.col.UpdateOne(ctx, filter, bson.D{{"$push", bson.D{{"users", u}}}})
	return err
}

func (rr MongoRoomRepository) LeaveRoom(ctx context.Context, userId int64, roomId string) error {
	hex, err := primitive.ObjectIDFromHex(roomId)
	if err != nil {
		return err
	}
	filter := bson.D{{"_id", bson.D{{"$eq", hex}}}}
	_, err = rr.col.UpdateOne(ctx, filter, bson.M{"$pull": bson.M{"users": bson.M{"_id": userId}}})
	if err != nil {
		return err
	}
	return nil
}

func (rr MongoRoomRepository) SaveRoom(ctx context.Context, r *api.Room) (primitive.ObjectID, error) {
	res, err := rr.col.InsertOne(ctx, r)
	if err != nil {
		log.Error().Err(err).Msg("insert failed")
	}
	if res != nil && res.InsertedID == nil {
		return primitive.NewObjectID(), errors.New("insert failed")
	}
	return res.InsertedID.(primitive.ObjectID), err
}

func (rr MongoRoomRepository) ArchiveRoom(ctx context.Context, userId int64, roomId string) error {
	hex, err := primitive.ObjectIDFromHex(roomId)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": hex, "users._id": userId}
	_, err = rr.col.UpdateOne(ctx, filter, bson.M{"$addToSet": bson.M{"room_states.archived": userId}})
	return err
}

func (rr MongoRoomRepository) UnArchiveRoom(ctx context.Context, userId int64, roomId string) error {
	hex, err := primitive.ObjectIDFromHex(roomId)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": hex, "users._id": userId}
	_, err = rr.col.UpdateOne(ctx, filter, bson.M{"$pull": bson.M{"room_states.archived": userId}})
	log.Error().Err(err).Msg("")
	return err
}

func (rr MongoRoomRepository) hasRoom(ctx context.Context, u *api.User) (bool, error) {
	resp, err := rr.col.CountDocuments(ctx, bson.D{{"_id", bson.D{{"$eq", u.ID}}}})
	return resp > 0, err
}

func (rr MongoRoomRepository) hasUserInRoom(ctx context.Context, uId int64, roomId primitive.ObjectID) (bool, error) {
	resp, err := rr.col.CountDocuments(ctx, bson.D{{"_id", bson.D{{"$eq", roomId}}},
		{"users._id", bson.D{{"$eq", uId}}}})
	return resp > 0, err
}

func (rr MongoRoomRepository) FindRoomsByUserId(ctx context.Context, userId int64) (*[]api.Room, error) {
	cur, err := rr.col.Find(ctx, bson.M{
		"users._id":            bson.M{"$eq": userId},
		"room_states.archived": bson.M{"$ne": userId},
	}, getOrderOptions("create_at", descParameter))
	if err != nil {
		return nil, err
	}
	var m []api.Room
	err = cur.All(ctx, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (rr MongoRoomRepository) FindArchivedRoomsByUserId(ctx context.Context, userId int64) (*[]api.Room, error) {
	cur, err := rr.col.Find(ctx, bson.M{
		"users._id":            bson.M{"$eq": userId},
		"room_states.archived": bson.M{"$eq": userId},
	}, getOrderOptions("create_at", descParameter))
	if err != nil {
		return nil, err
	}
	var m []api.Room
	err = cur.All(ctx, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (rr MongoRoomRepository) FindRoomsByLikeName(ctx context.Context, userId int64, name string) (*[]api.Room, error) {
	cur, err := rr.col.Find(ctx, bson.M{
		"users":                bson.M{"$elemMatch": bson.M{"_id": userId}},
		"name":                 bson.M{"$regex": ".*" + name + ".*"},
		"room_states.archived": bson.M{"$ne": userId},
	}, getOrderOptions("create_at", descParameter))
	if err != nil {
		return nil, err
	}
	var m []api.Room
	if err = cur.All(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func getOrderOptions(field string, orderParameter int) *options.FindOptions {
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{field, orderParameter}})
	return findOptions
}
