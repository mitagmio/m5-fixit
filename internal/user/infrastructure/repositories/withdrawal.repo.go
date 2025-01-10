package repositories

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Withdrawal represents a withdrawal record.
type Withdrawal struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Amount     float64            `bson:"amount" json:"amount"`
	Wallet     string             `bson:"wallet" json:"wallet"`
	JettonName string             `bson:"jetton_name,omitempty" json:"jetton_name,omitempty"`
	Status     string             `bson:"status" json:"status"`
	Timestamp  int64              `bson:"timestamp" json:"timestamp"`
}

// WithdrawalsRepository provides access to the withdrawals collection.
type WithdrawalsRepository struct {
	Collection *mongo.Collection
}

// NewWithdrawalsRepository creates a new WithdrawalsRepository.
func NewWithdrawalsRepository(db *mongo.Database) *WithdrawalsRepository {
	return &WithdrawalsRepository{
		Collection: db.Collection("withdrawals"),
	}
}

// CreateWithdrawal inserts a new withdrawal record into the database.
func (repo *WithdrawalsRepository) CreateWithdrawal(ctx context.Context, withdrawal *Withdrawal) error {

	// Mongo will automatically generate _id if it's empty
	_, err := repo.Collection.InsertOne(ctx, withdrawal)
	return err
}

// GetWithdrawalByID retrieves a withdrawal record by its ID.
func (repo *WithdrawalsRepository) GetWithdrawalByID(ctx context.Context, id string) (*Withdrawal, error) {
	var withdrawal Withdrawal
	// Convert string id to ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	err = repo.Collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&withdrawal)
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

// GetWithdrawalsByWallet retrieves all withdrawals for a specific wallet with an optional limit.
func (repo *WithdrawalsRepository) GetWithdrawalsByWallet(ctx context.Context, wallet string, limit int64) ([]Withdrawal, error) {
	filter := bson.M{"wallet": wallet}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	if limit > 0 {
		opts.SetLimit(limit)
	}

	cursor, err := repo.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var withdrawals []Withdrawal
	for cursor.Next(ctx) {
		var withdrawal Withdrawal
		if err := cursor.Decode(&withdrawal); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}

// GetLast50Withdrawals retrieves the last 50 withdrawals.
func (repo *WithdrawalsRepository) GetLast50Withdrawals(ctx context.Context) ([]Withdrawal, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(50)

	cursor, err := repo.Collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var withdrawals []Withdrawal
	for cursor.Next(ctx) {
		var withdrawal Withdrawal
		if err := cursor.Decode(&withdrawal); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}

// DeleteWithdrawal deletes a withdrawal record by its ID.
func (repo *WithdrawalsRepository) DeleteWithdrawal(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = repo.Collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

// GetLast50WithdrawalsWithJetton retrieves the last 50 withdrawals,
// optionally filtering by a specific JettonName.
func (repo *WithdrawalsRepository) GetLast50WithdrawalsWithJetton(ctx context.Context, jettonName string) ([]Withdrawal, error) {
	filter := bson.M{"jetton_name": bson.M{"$exists": true}} // Only retrieve withdrawals where jetton_name exists

	if jettonName != "" {
		filter["jetton_name"] = jettonName
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(50)

	cursor, err := repo.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var withdrawals []Withdrawal
	for cursor.Next(ctx) {
		var withdrawal Withdrawal
		if err := cursor.Decode(&withdrawal); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}

// GetLast50WithdrawalsWithoutJetton retrieves the last 50 withdrawals
func (repo *WithdrawalsRepository) GetLast50WithdrawalsWithoutJetton(ctx context.Context) ([]Withdrawal, error) {
	filter := bson.M{"jetton_name": bson.M{"$exists": false}}

	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(50)

	cursor, err := repo.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var withdrawals []Withdrawal
	for cursor.Next(ctx) {
		var withdrawal Withdrawal
		if err := cursor.Decode(&withdrawal); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}

// Создадим константы для статусов
const (
	StatusCreating              = "CREATING"
	StatusModerator             = "MODERATOR"
	StatusWithdrawing           = "WITHDRAWING"
	StatusWithdrawingAdmin      = "WITHDRAWING_ADMIN"
	StatusWithdrawingNow        = "WITHDAWING_NOW"
	StatusWithdrawingSuccess    = "WITHDRAWING_SUCCESS"
	StatusWithdrawingError      = "WITHDRAWING_ERROR"
	StatusWithdrawingErrorSeqno = "WITHDRAWING_ERROR_SEQNO"
	StatusCancelAdmin           = "CANCEL_ADMIN"
	StatusError                 = "ERROR"
)

func NewWithdrawal(wallet string, amount float64) *Withdrawal {
	return &Withdrawal{
		Wallet:    wallet,
		Amount:    amount,
		Status:    StatusCreating,
		Timestamp: time.Now().Unix(),
	}
}
