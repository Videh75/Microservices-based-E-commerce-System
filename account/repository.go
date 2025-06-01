//handles all the database operations

package account

import (
	"context"
	"database/sql"
)

type Repository interface {
	Close()
	PutAccount(ctx context.Context, a Account) error
	GetAccountByID(ctx context.Context, id string) (*Account, error)
	ListAccounts(ctx context.Context, skip uint64, take uint64) ([]Account, error)
}

// Internal implementation using *sql.DB
type postgresRepository struct {
	db *sql.DB
}

// Connects to PostgreSQL
// sql.Open() sets up the DB connection
// Ping() ensures the DB is alive
// Returns a postgresRepository that satisfies the Repository interface.
func NewPostgresRepository(url string) (Repository, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &postgresRepository{db}, nil
}

func (r *postgresRepository) Close() {
	r.db.Close()
}

func (r *postgresRepository) Ping() error {
	return r.db.Ping()
}

// Adds a new account into the database. $1, $2 are placeholders for a.id and a.name
func (r *postgresRepository) PutAccount(ctx context.Context, a Account) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO accounts(id, name) VALUES($1, $2)", a.ID, a.Name)
	return err
}

// Queries a single account by ID
func (r *postgresRepository) GetAccountByID(ctx context.Context, id string) (*Account, error) {
	row, err := r.db.QueryContext(ctx, "SELECT id, name FROM accounts WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	a := &Account{}
	if err := row.Scan(&a.ID, &a.Name); err != nil {
		return nil, err
	}
	return a, nil
}

// Returns a paginated list of accounts
func (r *postgresRepository) ListAccounts(ctx context.Context, skip uint64, take uint64) ([]Account, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name FROM accounts ORDER BY id DESC OFFSET $1 LIMIT $2", skip, take)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []Account{}

	for rows.Next() {
		a := &Account{}
		if err = rows.Scan(&a.ID, &a.Name); err == nil {
			accounts = append(accounts, *a)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}
