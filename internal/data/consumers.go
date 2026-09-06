package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/validator"
	"github.com/lib/pq"
)


var ErrDuplicateEmail = errors.New("duplicate email address")	
var ErrRecordNotFound = errors.New("no record found")
var ErrEditConflict = errors.New("edit conflict")

func ValidateConsumer(v *validator.Validator, consumer *Consumer) {
	v.Check(consumer.Name != "", "name", "must be provided")
	v.Check(len(consumer.Name) <= 200, "name", "must not exceed 200 characters")
	v.Check(consumer.Email != "", "email", "must be provided")
	v.Check(validator.Matches(consumer.Email, validator.EmailRX), "email", "must be a valid email address")
}

type Consumer struct {
	ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Status    string    `json:"status"`
    Version   int       `json:"version"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
	
}

type ConsumerModel struct {
	// This struct is used to interact with the database
	DB *sql.DB 
}


func (m ConsumerModel) Insert(c*Consumer) error {
	query := `
		INSERT INTO consumers (name, email)
		VALUES ($1, $2)
		RETURNING id, status, version, created_at, updated_at
		`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, c.Name, c.Email).Scan(
		&c.ID,
		&c.Status,
		&c.Version,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.Constraint == "consumers_email_key" {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}


func (m ConsumerModel) Get(id string) (*Consumer, error) {
	query := `
		SELECT id, name, email, status, version, created_at, updated_at
		FROM consumers
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var c Consumer

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.Status,
		&c.Version,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &c, nil
}	

func (m ConsumerModel) Update(c *Consumer) error {
	query := `
		UPDATE consumers
		SET name = $1, email = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version, updated_at
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, c.Name, c.Email, c.ID, c.Version).Scan(
		&c.Version,
		&c.UpdatedAt,
	)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.Constraint == "consumers_email_key" {
				return ErrDuplicateEmail
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	return nil
}		