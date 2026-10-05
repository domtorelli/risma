package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// recentMovementsLimit è il numero di movimenti mostrati (scheda, §4.2).
const recentMovementsLimit = 5

const totalUsersQuery = `
	SELECT count(*)
	FROM users
	WHERE deleted_at IS NULL`

// productTotalsQuery conta i prodotti a catalogo e ne calcola il valore in
// NUMERIC, restituito come testo per non passare da un float.
const productTotalsQuery = `
	SELECT count(*), COALESCE(SUM(stock_quantity * price_per_unit), 0.00)::text
	FROM products
	WHERE deleted_at IS NULL`

// recentMovementsQuery comprende anche i movimenti di prodotti eliminati:
// sono parte dello storico reale del magazzino.
const recentMovementsQuery = `
	SELECT m.id, m.movement_type, m.quantity, m.note, m.created_at,
		p.id, p.name, p.deleted_at IS NOT NULL, u.username
	FROM stock_movements m
	JOIN products p ON p.id = m.product_id
	JOIN users u ON u.id = m.user_id
	ORDER BY m.created_at DESC, m.id DESC
	LIMIT $1`

// productsByTypeQuery parte dai tipi con una LEFT JOIN, così anche un tipo
// senza prodotti compare con conteggio zero.
const productsByTypeQuery = `
	SELECT t.id, t.name, count(p.id)
	FROM product_types t
	LEFT JOIN products p ON p.product_type_id = t.id AND p.deleted_at IS NULL
	GROUP BY t.id, t.name
	ORDER BY t.name`

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Summary legge i dati della dashboard in una transazione in sola lettura
// con isolamento REPEATABLE READ: tutte le query vedono la stessa istantanea,
// quindi un movimento salvato nel frattempo non rende i dati incoerenti
// (es. valore dell'inventario e ultimi movimenti).
func (r *Repository) Summary(ctx context.Context) (Summary, error) {
	var s Summary

	opts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

	err := pgx.BeginTxFunc(ctx, r.pool, opts, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, totalUsersQuery).Scan(&s.TotalUsers); err != nil {
			return fmt.Errorf("counting users: %w", err)
		}

		if err := tx.QueryRow(ctx, productTotalsQuery).Scan(&s.TotalProducts, &s.InventoryValue); err != nil {
			return fmt.Errorf("computing product totals: %w", err)
		}

		rows, err := tx.Query(ctx, recentMovementsQuery, recentMovementsLimit)
		if err != nil {
			return fmt.Errorf("listing recent movements: %w", err)
		}

		s.RecentMovements, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Movement, error) {
			var m Movement
			err := row.Scan(&m.ID, &m.Type, &m.Quantity, &m.Note, &m.CreatedAt,
				&m.ProductID, &m.ProductName, &m.ProductDeleted, &m.Username)
			return m, err
		})
		if err != nil {
			return fmt.Errorf("reading recent movements: %w", err)
		}

		rows, err = tx.Query(ctx, productsByTypeQuery)
		if err != nil {
			return fmt.Errorf("counting products by type: %w", err)
		}

		s.ProductsByType, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (TypeCount, error) {
			var c TypeCount
			err := row.Scan(&c.TypeID, &c.TypeName, &c.Count)
			return c, err
		})
		if err != nil {
			return fmt.Errorf("reading products by type: %w", err)
		}

		return nil
	})
	if err != nil {
		return Summary{}, fmt.Errorf("reading dashboard: %w", err)
	}

	return s, nil
}
