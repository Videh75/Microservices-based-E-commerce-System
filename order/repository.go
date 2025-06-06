package order

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

type Repository interface {
	Close()
	PutOrder(ctx context.Context, o Order) error
	GetOrdersForAccount(ctx context.Context, accountID string) ([]Order, error)
}

type postgresRepository struct {
	db *sql.DB
}

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

// Inserts one row into orders and inserts N rows into order_products (for each product in the order)
// BeginTx() to execute multiple related queries, used for atomicity - if one query fails, we want to roll back everything.
// PrepareContext() to prepare a statement to efficiently batch insert many rows (ExecContext() will send each query separately — slower and more load on DB).
// pq.CopyIn allows fast bulk inserts
func (r *postgresRepository) PutOrder(ctx context.Context, o Order) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	tx.ExecContext(ctx, "INSERT INTO orders(id, created_at, account_id, total_price) VALUES ($1, $2, $3, $4)", o.ID, o.CreatedAt, o.AccountID, o.TotalPrice)
	if err != nil {
		return
	}
	stmt, _ := tx.PrepareContext(ctx, pq.CopyIn("order_products", "order_id", "product_id", "quantity"))
	for _, p := range o.Products {
		_, err = stmt.ExecContext(ctx, o.ID, p.ID, p.Quantity)
		if err != nil {
			return
		}
	}
	_, err = stmt.ExecContext(ctx) //  Once the stmt is prepared (it knows the table & columns), we only need to pass the context and values.
	if err != nil {
		return
	}
	stmt.Close() // because the prepared statement holds resources
	return
}

// Querying the orders and their associated products for a specific account.
// `total_price::money::numeric::float8` is used because PostgreSQL does not allow direct casting from MONEY to float8
// because MONEY includes formatting (like currency symbols, commas, etc.), and converting that directly to a float can cause errors or data loss.
func (r *postgresRepository) GetOrdersForAccount(ctx context.Context, accountID string) ([]Order, error) {
	// Query orders + their products with JOIN
	// The result will have 1 row per product per order (duplicate order data, different product_id per row)
	rows, err := r.db.QueryContext(ctx,
		`SELECT
		o.id,
		o.created_at,
		o.account_id,
		o.total_price::money::numeric::float8,
		op.product_id,
		op.quantity
		FROM orders o JOIN order_products op ON(o.id = op.order_id)
		WHERE o.account_id = $1
		GROUP BY o.id`,
		accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []Order{}
	order := &Order{}
	lastOrder := &Order{}
	orderedProduct := &OrderedProduct{}
	products := []OrderedProduct{}

	// Iterate through each row of the flattened JOIN result
	for rows.Next() {
		// Read current row's data into order + orderedProduct fields
		if err = rows.Scan(
			&order.ID,
			&order.CreatedAt,
			&order.AccountID,
			&order.TotalPrice,
			&orderedProduct.ID,
			&orderedProduct.Quantity,
		); err != nil {
			return nil, err
		}
		// Each product of an order comes as a separate row in the JOIN result.
		// So for the same order_id, multiple rows will appear — each with a different product_id and quantity.
		// As long as we see the same order_id, we keep appending the products to the 'products' slice.
		// When we detect that the order_id has changed (a new order is starting),
		// we "flush" the current order:
		//   - create an Order object with its list of collected products,
		//   - append it to the 'orders' slice,
		//   - and reset 'products' to start collecting products for the next order.
		if lastOrder.ID != "" && lastOrder.ID != order.ID {
			neworder := Order{
				ID:         lastOrder.ID,
				CreatedAt:  lastOrder.CreatedAt,
				AccountID:  lastOrder.AccountID,
				TotalPrice: lastOrder.TotalPrice,
				Products:   products,
			}
			orders = append(orders, neworder)
			products = []OrderedProduct{} // reset products slice for new order
		}

		// Always append the current product to the products slice
		products = append(products, OrderedProduct{
			ID:       orderedProduct.ID,
			Quantity: orderedProduct.Quantity,
		})
		*lastOrder = *order // Remember current order as lastOrder for next iteration comparison
	}

	// After the loop, flush the last order (since it won't be flushed in the loop)
	if lastOrder != nil {
		neworder := Order{
			ID:         lastOrder.ID,
			CreatedAt:  lastOrder.CreatedAt,
			AccountID:  lastOrder.AccountID,
			TotalPrice: lastOrder.TotalPrice,
			Products:   lastOrder.Products,
		}
		orders = append(orders, neworder)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}
