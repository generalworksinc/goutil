package gw_gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"gorm.io/gorm"
)

// WriteTxOptions は WriteTransactionWithOptions の SQLITE_BUSY リトライ設定。
// ゼロ値のフィールドは DefaultWriteTxOptions の値で補われる。
type WriteTxOptions struct {
	// MaxAttempts は BUSY 時にトランザクション全体をやり直す最大回数(初回を含む)。
	MaxAttempts int
	// BaseBackoff は1回目のリトライ前の待ち時間。以降は倍々に伸び、同程度までのジッタが加わる。
	BaseBackoff time.Duration
	// MaxBackoff は1回あたりの待ち時間(ジッタ除く)の上限。
	MaxBackoff time.Duration
}

// DefaultWriteTxOptions は既定値(6回、25ms → 最大500ms、合計1.5秒程度)を返す。
func DefaultWriteTxOptions() WriteTxOptions {
	return WriteTxOptions{MaxAttempts: 6, BaseBackoff: 25 * time.Millisecond, MaxBackoff: 500 * time.Millisecond}
}

func (o WriteTxOptions) withDefaults() WriteTxOptions {
	d := DefaultWriteTxOptions()
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = d.MaxAttempts
	}
	if o.BaseBackoff <= 0 {
		o.BaseBackoff = d.BaseBackoff
	}
	if o.MaxBackoff <= 0 {
		o.MaxBackoff = d.MaxBackoff
	}
	return o
}

// WriteTransaction は書き込みを含むトランザクションを開始する。
//
// SQLite 系(ローカル SQLite / Turso libsql など dialect 名が "sqlite" のもの)では、素の BEGIN(DEFERRED) だと
// 「読んでから書く」トランザクションが書き込みロックへ昇格する瞬間に他の書き込みと衝突すると、
// 待たずに SQLITE_BUSY("database is locked") で失敗する。そのため BEGIN IMMEDIATE で開始して最初から
// 書き込みロックを取り、それでも BUSY になった場合は短いバックオフを挟んでトランザクション全体をやり直す。
// SQLite 系以外(PostgreSQL / MySQL など)は db.Transaction と同じ挙動。
//
// db が既にトランザクション中(ConnPool が TxCommitter)の場合は db.Transaction に委譲し、
// GORM 標準どおり SAVEPOINT によるネストになる。
//
// fn はやり直されることがあるため、DB 外への副作用(通知送信など)を持たせないこと。
func WriteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return WriteTransactionWithOptions(db, DefaultWriteTxOptions(), fn)
}

// WriteTransactionWithOptions は WriteTransaction のリトライ設定を指定できる版。
func WriteTransactionWithOptions(db *gorm.DB, opts WriteTxOptions, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	if db.Error != nil {
		return db.Error
	}
	if !IsSQLiteDialect(db) || isInsideTransaction(db) {
		return db.Transaction(fn)
	}
	opts = opts.withDefaults()
	return retryOnSQLiteBusy(statementContext(db), opts, func() error {
		return runImmediateTx(db, fn)
	})
}

// IsSQLiteDialect は db の dialect が SQLite 系(gorm.io/driver/sqlite、glebarez/sqlite、libsql 経由も含む)かを返す。
func IsSQLiteDialect(db *gorm.DB) bool {
	return db != nil && db.Dialector != nil && db.Dialector.Name() == "sqlite"
}

// IsSQLiteBusyError は SQLite の SQLITE_BUSY 系エラーかどうかを判定する。
// mattn/go-sqlite3 と glebarez/sqlite は "database is locked"、Turso(libsql) は
// "SQLite error: database is locked" の文字列で返る。
func IsSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "sqlite_busy")
}

func isInsideTransaction(db *gorm.DB) bool {
	committer, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok && committer != nil
}

func statementContext(db *gorm.DB) context.Context {
	if db.Statement != nil && db.Statement.Context != nil {
		return db.Statement.Context
	}
	return context.Background()
}

// retryOnSQLiteBusy は fn が BUSY エラーで失敗した場合に限り、バックオフ付きでやり直す。
func retryOnSQLiteBusy(ctx context.Context, opts WriteTxOptions, fn func() error) error {
	var err error
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		err = fn()
		if err == nil || !IsSQLiteBusyError(err) {
			return err
		}
		if attempt == opts.MaxAttempts {
			break
		}
		wait := opts.backoff(attempt)
		slog.WarnContext(ctx, "write transaction hit SQLITE_BUSY; retrying",
			slog.Int("attempt", attempt), slog.Duration("wait", wait), slog.String("error", err.Error()))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return err
		case <-timer.C:
		}
	}
	slog.ErrorContext(ctx, "write transaction gave up after SQLITE_BUSY retries",
		slog.Int("attempts", opts.MaxAttempts), slog.String("error", err.Error()))
	return err
}

// backoff は BaseBackoff, 2倍, 4倍, ... と伸ばし(MaxBackoff 上限)、同程度までのジッタを足した待ち時間を返す。
func (o WriteTxOptions) backoff(attempt int) time.Duration {
	base := o.BaseBackoff
	for i := 1; i < attempt && base < o.MaxBackoff; i++ {
		base *= 2
	}
	if base > o.MaxBackoff {
		base = o.MaxBackoff
	}
	return base + time.Duration(rand.Int64N(int64(base)))
}

// immediateTx は接続プールから固定した1本の接続を、GORM から見て「トランザクション中の接続」として
// 扱うための薄いラッパー。TxCommitter(Commit/Rollback) を実装しているため、fn の中で
// tx.Transaction(...) を呼んだ場合は GORM 標準の挙動どおり SAVEPOINT によるネストになる。
//
// *sql.Conn を埋め込まず各メソッドを明示的に委譲しているのは意図的。GORM の契約として、トランザクション中の
// ConnPool は BeginTx を持ってはならない。埋め込むと *sql.Conn の BeginTx が公開され、GORM が Create/Save/Delete の
// 既定トランザクション(gorm:begin_transaction)で TxBeginner と判定して同じ接続に BEGIN を重ねてしまう
// (SQLite では "cannot start a transaction within a transaction"、PostgreSQL では警告のみで内側の COMMIT が
// 外側を途中コミット、MySQL では暗黙コミット)。*sql.Tx と同じく BeginTx を持たないことで
// ErrInvalidTransaction → スキップされる。
type immediateTx struct {
	conn *sql.Conn
}

var (
	_ gorm.ConnPool    = (*immediateTx)(nil)
	_ gorm.TxCommitter = (*immediateTx)(nil)
)

func (t *immediateTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return t.conn.PrepareContext(ctx, query)
}

func (t *immediateTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.conn.ExecContext(ctx, query, args...)
}

func (t *immediateTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.conn.QueryContext(ctx, query, args...)
}

func (t *immediateTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.conn.QueryRowContext(ctx, query, args...)
}

func (t *immediateTx) Commit() error {
	_, err := t.conn.ExecContext(context.Background(), "COMMIT")
	return err
}

func (t *immediateTx) Rollback() error {
	_, err := t.conn.ExecContext(context.Background(), "ROLLBACK")
	return err
}

// runImmediateTx は BEGIN IMMEDIATE で1回分のトランザクションを実行する。
// 失敗時は必ず ROLLBACK し、ROLLBACK 自体に失敗した接続はプールへ戻さず破棄する
// (トランザクションを握ったままの接続がプールに戻ると、以後の書き込みが全てロックされるため)。
func runImmediateTx(db *gorm.DB, fn func(tx *gorm.DB) error) (err error) {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx := statementContext(db)
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	itx := &immediateTx{conn: conn}
	tx := db.Session(&gorm.Session{Context: ctx, NewDB: true})
	tx.Statement.ConnPool = itx

	if err = tx.Exec("BEGIN IMMEDIATE").Error; err != nil {
		return err
	}

	finished := false
	defer func() {
		if finished {
			return
		}
		// fn 内の panic、fn のエラー、COMMIT 失敗のいずれでもここに来る。
		if rollbackErr := itx.Rollback(); rollbackErr != nil {
			slog.ErrorContext(ctx, "rollback failed; discarding connection", slog.String("error", rollbackErr.Error()))
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	if err = itx.Commit(); err != nil {
		return err
	}
	finished = true
	return nil
}
