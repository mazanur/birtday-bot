package repository

import (
	"context"
	"github.com/almaznur91/splitty/internal/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository interface {
	UpsertUser(ctx context.Context, u api.User) (*api.User, error)
	SetUserLang(ctx context.Context, userId int64, lang string) error
	SetNotificationUser(ctx context.Context, userId int64, notification bool) error
	SetCountInPage(ctx context.Context, userId int64, count int) error
	FindById(ctx context.Context, id int64) (*api.User, error)
}

type ChatStateRepository interface {
	Save(ctx context.Context, u *api.ChatState) error
	FindById(ctx context.Context, id int) (*api.ChatState, error)
	FindByUserId(ctx context.Context, userId int64) (*api.ChatState, error)
	DeleteById(ctx context.Context, id primitive.ObjectID) error
	DeleteByUserId(ctx context.Context, id int64) error
}

type ButtonRepository interface {
	Save(ctx context.Context, b *api.Button) (primitive.ObjectID, error)
	SaveAll(ctx context.Context, b ...*api.Button) ([]*api.Button, error)
	FindById(ctx context.Context, id string) (*api.Button, error)
}

type MongoUserRepository struct {
	col *mongo.Collection
}

type MongoChatStateRepository struct {
	col *mongo.Collection
}
type MongoButtonRepository struct {
	col *mongo.Collection
}

func NewUserRepository(col *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{col: col.Collection("user")}
}

func NewChatStateRepository(col *mongo.Database) *MongoChatStateRepository {
	return &MongoChatStateRepository{col: col.Collection("chat_state")}
}

func NewButtonRepository(col *mongo.Database) *MongoButtonRepository {
	return &MongoButtonRepository{col: col.Collection("button")}
}

func (r MongoUserRepository) FindById(ctx context.Context, id int64) (*api.User, error) {
	res := r.col.FindOne(ctx, bson.D{{"_id", bson.D{{"$eq", id}}}})
	if res.Err() != nil {
		return nil, res.Err()
	}
	cs := &api.User{}
	if err := res.Decode(cs); err != nil {
		return nil, err
	}
	if cs.CountInPage == 0 {
		cs.CountInPage = 5
	}
	if cs.NotificationOn == nil {
		cs.NotificationOn = func() *bool { b := true; return &b }()
	}
	return cs, nil
}

func (r MongoUserRepository) UpsertUser(ctx context.Context, u api.User) (*api.User, error) {
	opts := options.Update().SetUpsert(true)
	f := bson.D{{"_id", bson.D{{"$eq", u.ID}}}}
	update := bson.D{{"$set", bson.M{"_id": u.ID, "user_lang": u.UserLang, "display_name": u.DisplayName, "user_name": u.Username}}}
	_, err := r.col.UpdateOne(ctx, f, update, opts)
	if err != nil {
		return nil, err
	}
	return r.FindById(ctx, u.ID)
}

func (r MongoUserRepository) SetUserLang(ctx context.Context, userId int64, lang string) error {
	opts := options.Update().SetUpsert(true)
	f := bson.D{{"_id", bson.D{{"$eq", userId}}}}
	update := bson.D{{"$set", bson.M{"selected_lang": lang}}}
	_, err := r.col.UpdateOne(ctx, f, update, opts)
	if err != nil {
		return err
	}
	return nil
}

func (r MongoUserRepository) SetCountInPage(ctx context.Context, userId int64, count int) error {
	opts := options.Update().SetUpsert(true)
	f := bson.D{{"_id", bson.D{{"$eq", userId}}}}
	update := bson.D{{"$set", bson.M{"count_in_page": count}}}
	_, err := r.col.UpdateOne(ctx, f, update, opts)
	if err != nil {
		return err
	}
	return nil
}

func (r MongoUserRepository) SetNotificationUser(ctx context.Context, userId int64, notification bool) error {
	opts := options.Update().SetUpsert(true)
	f := bson.D{{"_id", bson.D{{"$eq", userId}}}}
	update := bson.D{{"$set", bson.M{"notification_on": notification}}}
	_, err := r.col.UpdateOne(ctx, f, update, opts)
	if err != nil {
		return err
	}
	return nil
}

func (csr MongoChatStateRepository) Save(ctx context.Context, cs *api.ChatState) error {
	res, err := csr.col.InsertOne(ctx, cs)
	if err != nil {
		log.Error().Err(err).Msg("insert failed")
	}
	if res != nil && res.InsertedID == nil {
		return errors.New("insert failed")
	}
	return err
}

func (csr MongoChatStateRepository) FindById(ctx context.Context, id int) (*api.ChatState, error) {
	res := csr.col.FindOne(ctx, bson.D{{"_id", bson.D{{"$eq", id}}}})
	if res.Err() == mongo.ErrNoDocuments {
		log.Warn().Err(res.Err()).Msgf("chat_state not found by id %v", id)
		return nil, nil
	}
	if res.Err() != nil {
		return nil, res.Err()
	}
	cs := &api.ChatState{}
	if err := res.Decode(cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (csr MongoChatStateRepository) FindByUserId(ctx context.Context, userId int64) (*api.ChatState, error) {
	res := csr.col.FindOne(ctx, bson.D{{"user_id", bson.D{{"$eq", userId}}}})
	if res.Err() == mongo.ErrNoDocuments {
		log.Debug().Err(res.Err()).Msgf("chat_state not found by user_id %v", userId)
		return nil, nil
	}
	if res.Err() != nil {
		return nil, res.Err()
	}
	cs := &api.ChatState{}
	if err := res.Decode(cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (csr MongoChatStateRepository) DeleteById(ctx context.Context, id primitive.ObjectID) error {
	_, err := csr.col.DeleteOne(ctx, bson.D{{"_id", bson.D{{"$eq", id}}}})
	if err != nil {
		log.Error().Err(err).Msg("delete failed")
		return err
	}
	return nil
}

func (csr MongoChatStateRepository) DeleteByUserId(ctx context.Context, id int64) error {
	if _, err := csr.col.DeleteMany(ctx, bson.M{"user_id": id}); err != nil {
		log.Error().Err(err).Msg("delete failed")
		return err
	}
	return nil
}

func (br MongoButtonRepository) Save(ctx context.Context, b *api.Button) (primitive.ObjectID, error) {
	res, err := br.col.InsertOne(ctx, b)
	if err != nil || res == nil || res.InsertedID == nil {
		log.Error().Err(err).Stack().Msg("insert failed")
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func (br MongoButtonRepository) SaveAll(ctx context.Context, b ...*api.Button) ([]*api.Button, error) {
	i := make([]interface{}, len(b))
	for idx, btn := range b {
		i[idx] = btn
	}
	res, err := br.col.InsertMany(ctx, i)
	if err != nil || res == nil || res.InsertedIDs == nil {
		log.Error().Err(err).Stack().Msg("insert failed")
		return b, err
	}
	for idx, id := range res.InsertedIDs {
		b[idx].ID = id.(primitive.ObjectID)
	}

	return b, nil
}

func (br MongoButtonRepository) FindById(ctx context.Context, id string) (*api.Button, error) {
	hex, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	res := br.col.FindOne(ctx, bson.M{"_id": hex})
	if res.Err() != nil {
		return nil, res.Err()
	}
	btn := &api.Button{}
	if err = res.Decode(btn); err != nil {
		return nil, err
	}
	return btn, nil
}
