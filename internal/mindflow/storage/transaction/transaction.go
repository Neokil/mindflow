// Package transaction supplies the common SQLite transaction handle used across domain storage adapters.
package transaction

import "database/sql"

// Transaction is an atomic SQLite unit of work; its database behavior stays inside storage adapters.
type Transaction = sql.Tx
