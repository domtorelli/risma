package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/matteotorelli/risma/backend/internal/database"
)

// Codici SQLSTATE di PostgreSQL.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
	checkViolation      = "23514"
	numericOutOfRange   = "22003"
)

// Nomi dei vincoli creati dalla migrazione 00005 (nomi generati da
// PostgreSQL). stockCheck (stock_quantity >= 0) è l'ultima difesa contro
// una giacenza negativa.
const (
	stockCheck = "products_stock_quantity_check"
	typeFKey   = "products_product_type_id_fkey"
)

var (
	ErrNotFound          = errors.New("product not found")
	ErrAlreadyExists     = errors.New("a product with this name already exists")
	ErrInvalidType       = errors.New("product type does not exist")
	ErrInsufficientStock = errors.New("not enough stock for this movement")
	ErrQuantityTooLarge  = errors.New("resulting stock quantity is too large")
	ErrStockNotEmpty     = errors.New("product still has stock")
)

// productColumns elenca le colonne nello stesso ordine in cui scanProduct
// le legge. Gli alias sono p (products), t (product_types), c (users,
// l'utente che ha creato il prodotto). Il prezzo è letto come testo nel
// formato canonico di PostgreSQL (es. "12.50").
const productColumns = `p.id, p.creator_id, c.username, p.product_type_id, t.name,
	p.name, p.description, p.price_per_unit::text,
	p.stock_quantity, p.min_stock_threshold,
	p.created_at, p.updated_at, p.deleted_at`

const productJoins = `
	JOIN product_types t ON t.id = p.product_type_id
	JOIN users c ON c.id = p.creator_id`

const findByIDQuery = `
	SELECT ` + productColumns + `
	FROM products p ` + productJoins + `
	WHERE p.id = $1 AND p.deleted_at IS NULL`

// filterCondition seleziona i prodotti non eliminati il cui nome
// corrisponde al pattern ILIKE $1, del tipo $2 se non è NULL.
const filterCondition = `
	p.deleted_at IS NULL
	AND p.name ILIKE $1
	AND ($2::uuid IS NULL OR p.product_type_id = $2)`

// listQueryTemplate ha un segnaposto %s per l'ordinamento, che viene
// sostituito solo con un valore di orderClauses: mai con l'input dell'utente.
const listQueryTemplate = `
	SELECT ` + productColumns + `
	FROM products p ` + productJoins + `
	WHERE ` + filterCondition + `
	ORDER BY %s, p.id
	LIMIT $3 OFFSET $4`

const countQuery = `
	SELECT count(*)
	FROM products p
	WHERE ` + filterCondition

// orderClauses associa i criteri di ordinamento ammessi alla clausola SQL.
var orderClauses = map[string]string{
	"name asc":      "lower(p.name) ASC",
	"name desc":     "lower(p.name) DESC",
	"price asc":     "p.price_per_unit ASC",
	"price desc":    "p.price_per_unit DESC",
	"quantity asc":  "p.stock_quantity ASC",
	"quantity desc": "p.stock_quantity DESC",
}

const defaultOrder = "name asc"

const listTypesQuery = `
	SELECT id, name, description
	FROM product_types
	ORDER BY name`

const insertProductQuery = `
	INSERT INTO products (creator_id, product_type_id, name, description,
		price_per_unit, stock_quantity, min_stock_threshold)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id`

const insertMovementQuery = `
	INSERT INTO stock_movements (product_id, user_id, movement_type, quantity, note)
	VALUES ($1, $2, $3, $4, $5)`

// applyMovementQuery aggiorna la giacenza di un prodotto non eliminato.
// L'UPDATE blocca la riga fino alla fine della transazione, quindi due
// movimenti concorrenti sullo stesso prodotto vengono applicati uno dopo
// l'altro e il CHECK sulla giacenza vede sempre il valore aggiornato.
const applyMovementQuery = `
	UPDATE products
	SET stock_quantity = stock_quantity + $2
	WHERE id = $1 AND deleted_at IS NULL
	RETURNING id`

const updateQuery = `
	WITH p AS (
		UPDATE products
		SET product_type_id = $2, name = $3, description = $4,
			price_per_unit = $5, min_stock_threshold = $6
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *
	)
	SELECT ` + productColumns + `
	FROM p ` + productJoins

// softDeleteQuery elimina il prodotto solo se la giacenza è zero. L'UPDATE
// blocca la riga e, se un movimento concorrente l'ha appena modificata,
// ricontrolla la condizione sul valore aggiornato: un carico arrivato nel
// frattempo impedisce l'eliminazione.
const softDeleteQuery = `
	UPDATE products
	SET deleted_at = now()
	WHERE id = $1 AND deleted_at IS NULL AND stock_quantity = 0`

const stockQuantityQuery = `
	SELECT stock_quantity
	FROM products
	WHERE id = $1 AND deleted_at IS NULL`

// ListFilter contiene i criteri dell'elenco prodotti. Search e TypeID vuoti
// non filtrano; Order è una chiave di orderClauses (es. "price desc").
type ListFilter struct {
	Search string
	TypeID string
	Order  string
	Limit  int
	Offset int
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// scanProduct legge una riga con le colonne di productColumns.
func scanProduct(row pgx.Row) (Product, error) {
	var p Product

	err := row.Scan(
		&p.ID,
		&p.CreatorID,
		&p.CreatorUsername,
		&p.TypeID,
		&p.TypeName,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.StockQuantity,
		&p.MinStockThreshold,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		return Product{}, err
	}

	return p, nil
}

// knownError traduce un errore di scrittura nell'errore del pacchetto che lo
// descrive, oppure restituisce nil se l'errore è imprevisto.
func knownError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == uniqueViolation:
			return ErrAlreadyExists
		case pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == typeFKey:
			return ErrInvalidType
		case pgErr.Code == checkViolation && pgErr.ConstraintName == stockCheck:
			return ErrInsufficientStock
		case pgErr.Code == numericOutOfRange:
			return ErrQuantityTooLarge
		}
	}

	return nil
}

// typeFilter restituisce il parametro del filtro per tipo: NULL se vuoto.
func typeFilter(typeID string) *string {
	if typeID == "" {
		return nil
	}

	return &typeID
}

// ListTypes restituisce i tipi di prodotto ordinati per nome.
func (r *Repository) ListTypes(ctx context.Context) ([]Type, error) {
	rows, err := r.pool.Query(ctx, listTypesQuery)
	if err != nil {
		return nil, fmt.Errorf("listing product types: %w", err)
	}

	types, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Type, error) {
		var t Type
		err := row.Scan(&t.ID, &t.Name, &t.Description)
		return t, err
	})
	if err != nil {
		return nil, fmt.Errorf("reading product types: %w", err)
	}

	return types, nil
}

// FindByID cerca un prodotto non eliminato.
func (r *Repository) FindByID(ctx context.Context, id string) (Product, error) {
	p, err := scanProduct(r.pool.QueryRow(ctx, findByIDQuery, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrNotFound
		}

		return Product{}, fmt.Errorf("finding product by id: %w", err)
	}

	return p, nil
}

// List restituisce una pagina di prodotti non eliminati secondo il filtro.
func (r *Repository) List(ctx context.Context, f ListFilter) ([]Product, error) {
	orderBy, ok := orderClauses[f.Order]
	if !ok {
		orderBy = orderClauses[defaultOrder]
	}

	query := fmt.Sprintf(listQueryTemplate, orderBy)

	rows, err := r.pool.Query(ctx, query, database.ContainsPattern(f.Search), typeFilter(f.TypeID), f.Limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}

	products, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Product, error) {
		return scanProduct(row)
	})
	if err != nil {
		return nil, fmt.Errorf("reading products: %w", err)
	}

	return products, nil
}

// Count conta i prodotti che List restituirebbe senza paginazione.
func (r *Repository) Count(ctx context.Context, f ListFilter) (int, error) {
	var total int

	err := r.pool.QueryRow(ctx, countQuery, database.ContainsPattern(f.Search), typeFilter(f.TypeID)).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("counting products: %w", err)
	}

	return total, nil
}

// Create inserisce un prodotto con la giacenza iniziale p.StockQuantity e,
// se è positiva, registra il carico iniziale con la nota indicata. Prodotto
// e movimento sono scritti nella stessa transazione: o entrambi o nessuno.
// Usa i campi CreatorID, TypeID, Name, Description, Price, StockQuantity e
// MinStockThreshold.
func (r *Repository) Create(ctx context.Context, p Product, initialNote string) (Product, error) {
	var created Product

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var id string

		err := tx.QueryRow(ctx, insertProductQuery,
			p.CreatorID,
			p.TypeID,
			p.Name,
			p.Description,
			p.Price,
			p.StockQuantity,
			p.MinStockThreshold,
		).Scan(&id)
		if err != nil {
			return err
		}

		if p.StockQuantity > 0 {
			_, err := tx.Exec(ctx, insertMovementQuery, id, p.CreatorID, MovementIn, p.StockQuantity, initialNote)
			if err != nil {
				return err
			}
		}

		created, err = scanProduct(tx.QueryRow(ctx, findByIDQuery, id))
		return err
	})
	if err != nil {
		if known := knownError(err); known != nil {
			return Product{}, known
		}

		return Product{}, fmt.Errorf("creating product: %w", err)
	}

	return created, nil
}

// Update modifica i dati di un prodotto non eliminato. Usa i campi ID,
// TypeID, Name, Description, Price e MinStockThreshold: la giacenza cambia
// solo con RecordMovement.
func (r *Repository) Update(ctx context.Context, p Product) (Product, error) {
	updated, err := scanProduct(r.pool.QueryRow(ctx, updateQuery,
		p.ID,
		p.TypeID,
		p.Name,
		p.Description,
		p.Price,
		p.MinStockThreshold,
	))
	if err != nil {
		if known := knownError(err); known != nil {
			return Product{}, known
		}

		return Product{}, fmt.Errorf("updating product: %w", err)
	}

	return updated, nil
}

// RecordMovement applica un carico o scarico alla giacenza e registra il
// movimento nella stessa transazione. Restituisce il prodotto aggiornato,
// ErrNotFound se il prodotto non esiste o è eliminato, ErrInsufficientStock
// se uno scarico porterebbe la giacenza sotto zero.
func (r *Repository) RecordMovement(ctx context.Context, m Movement) (Product, error) {
	var updated Product

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var id string

		if err := tx.QueryRow(ctx, applyMovementQuery, m.ProductID, m.delta()).Scan(&id); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, insertMovementQuery, m.ProductID, m.UserID, m.Type, m.Quantity, m.Note)
		if err != nil {
			return err
		}

		updated, err = scanProduct(tx.QueryRow(ctx, findByIDQuery, id))
		return err
	})
	if err != nil {
		if known := knownError(err); known != nil {
			return Product{}, known
		}

		return Product{}, fmt.Errorf("recording stock movement: %w", err)
	}

	return updated, nil
}

// SoftDelete elimina logicamente un prodotto con giacenza zero; i suoi
// movimenti restano. Restituisce ErrStockNotEmpty se la giacenza è positiva,
// ErrNotFound se il prodotto non esiste o è già eliminato.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, softDeleteQuery, id)
	if err != nil {
		return fmt.Errorf("deleting product: %w", err)
	}

	if tag.RowsAffected() == 1 {
		return nil
	}

	// Nessuna riga eliminata: il prodotto non esiste oppure ha giacenza.
	var quantity int
	if err := r.pool.QueryRow(ctx, stockQuantityQuery, id).Scan(&quantity); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}

		return fmt.Errorf("checking product stock: %w", err)
	}

	return ErrStockNotEmpty
}
