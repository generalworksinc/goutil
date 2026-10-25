package gw_gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log/slog"
	"math/rand/v2"
	"regexp"
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
// fn はやり直されることがあるため、次を守ること。
//   - DB 外への副作用(通知送信など)を持たせない
//   - 書き込む値(モデルなど)は fn の中で作る。DB がトランザクションを取り消しても Go の値は戻らない
//     (fn の外で作ったモデルには、取り消された INSERT の ID などが残り、やり直しで別の SQL になる)
//   - fn の外の変数(集計や戻り値)には、WriteTransaction が成功した後に結果を入れる
//
// UseWriteRetry を呼んだ DB では、そこで指定したやり直しの設定を使う。
func WriteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	opts := DefaultWriteTxOptions()
	if p := writeRetryPluginOf(db); p != nil {
		opts = p.retry
	}
	return WriteTransactionWithOptions(db, opts, fn)
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
	begin := "BEGIN IMMEDIATE"
	if p := writeRetryPluginOf(db); p != nil && p.concurrent {
		begin = "BEGIN CONCURRENT"
	}
	return retryWrite(statementContext(db), opts, func() error {
		err := runWriteTx(db, begin, fn)
		if begin == "BEGIN CONCURRENT" && needsExclusiveTx(err) {
			// DDL は BEGIN CONCURRENT の中で使えない。このトランザクションは BEGIN IMMEDIATE でやり直す。
			begin = "BEGIN IMMEDIATE"
			return runWriteTx(db, begin, fn)
		}
		return err
	})
}

// IsSQLiteDialect は db の dialect が SQLite 系(gorm.io/driver/sqlite、glebarez/sqlite、libsql 経由も含む)かを返す。
func IsSQLiteDialect(db *gorm.DB) bool {
	return db != nil && db.Dialector != nil && db.Dialector.Name() == "sqlite"
}

// IsSQLiteBusyError は SQLite の SQLITE_BUSY 系エラーかどうかを判定する。
// mattn/go-sqlite3 と glebarez/sqlite は "database is locked"、Turso(libsql) は
// "SQLite error: database is locked"、Turso Database は "database is busy" の文字列で返る。
func IsSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "database is busy") ||
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

// retryWrite は fn がやり直せるエラー(isRetryableWriteError)で失敗した場合に限り、バックオフ付きでやり直す。
func retryWrite(ctx context.Context, opts WriteTxOptions, fn func() error) error {
	var err error
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		err = fn()
		if err == nil || !isRetryableWriteError(err) {
			return err
		}
		if attempt == opts.MaxAttempts {
			break
		}
		wait := opts.backoff(attempt)
		slog.WarnContext(ctx, "write transaction hit a retryable error; retrying",
			slog.Int("attempt", attempt), slog.Duration("wait", wait), slog.String("error", err.Error()))
		if !sleepCtx(ctx, wait) {
			return err
		}
	}
	slog.ErrorContext(ctx, "write transaction gave up after retries",
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
	// aborted は DB が取り消したトランザクション(Turso の MVCC の衝突)のエラー。取り消された後の SQL は
	// トランザクションの外で確定してしまうため、以後の SQL と COMMIT はすべてこのエラーで拒否する
	// (内側の Transaction のエラーを fn が握りつぶして続けても、一部だけが残らないように)。
	aborted error
	// concurrent は BEGIN CONCURRENT で始めたトランザクションか。衝突で取り消されうるのはこちらだけ。
	concurrent bool
	// covered は UseWriteRetry の callback(gorm:create など)の中で実行中か。その中の QueryContext の衝突は
	// callback が db.Error を見て拾える。それ以外の経路(Raw(...).Scan / Rows / Row)は行を読む処理が callback の外に
	// あって衝突を拾えないため、BEGIN CONCURRENT の中では書き込みの SQL を通さない。
	covered int
}

var errUncoveredWriteQuery = errors.New("gw_gorm: BEGIN CONCURRENT のトランザクションでは、Raw(...).Scan / Rows / Row で書き込みの SQL を流せない(衝突を検知できないため)。Create / Update / Delete か Exec を使う")

// rejectQuery は QueryContext / QueryRowContext で流してはいけない SQL ならエラーを返す。
func (t *immediateTx) rejectQuery(query string) error {
	if t.aborted != nil {
		return t.aborted
	}
	if t.concurrent && t.covered == 0 && isWriteSQL(query) {
		return errUncoveredWriteQuery
	}
	return nil
}

// isWriteSQL は SQL が書き込みになりうるかを返す。SELECT・WITH・EXPLAIN・VALUES、値を設定しない PRAGMA のどれかで始まり、
// INSERT / UPDATE / DELETE / REPLACE の語を含まないものだけを読み取りとする。
// shortcut: 先頭のコメントや、文字列の中のこれらの語も書き込みとみなす(止める側に倒れるだけなので安全)。読み取りの Raw が
// 止められて困る例が出たら、字句を解析する判定にする。
func isWriteSQL(query string) bool {
	q := strings.ToUpper(strings.TrimSpace(query))
	readOnly := readOnlyStart.MatchString(q) && !dmlWord.MatchString(q) &&
		!(strings.HasPrefix(q, "PRAGMA") && strings.Contains(q, "="))
	return !readOnly
}

var (
	readOnlyStart = regexp.MustCompile(`^(SELECT|WITH|EXPLAIN|VALUES|PRAGMA)\b`)
	dmlWord       = regexp.MustCompile(`\b(INSERT|UPDATE|DELETE|REPLACE)\b`)
)

// markAborted は衝突のエラーを受けたら、このトランザクションを取り消し済みにする。
func (t *immediateTx) markAborted(err error) {
	if t.aborted == nil && isWriteConflict(err) {
		t.aborted = err
	}
}

var (
	_ gorm.ConnPool    = (*immediateTx)(nil)
	_ gorm.TxCommitter = (*immediateTx)(nil)
)

func (t *immediateTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if t.aborted != nil {
		return nil, t.aborted
	}
	return t.conn.PrepareContext(ctx, query)
}

func (t *immediateTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if t.aborted != nil {
		return nil, t.aborted
	}
	res, err := t.conn.ExecContext(ctx, query, args...)
	t.markAborted(err)
	return res, err
}

// QueryContext の衝突(RETURNING 付きの書き込み)は行を読む途中で返るため、ここでは拾えない。
// UseWriteRetry の callback が db.Error を見て markAborted する。
func (t *immediateTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if err := t.rejectQuery(query); err != nil {
		return nil, err
	}
	rows, err := t.conn.QueryContext(ctx, query, args...)
	t.markAborted(err)
	return rows, err
}

func (t *immediateTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if err := t.rejectQuery(query); err != nil {
		// *sql.Row にエラーを持たせる手段が無いので、取り消し済みの context で流して SQL を実行させない
		// (Row のエラーは context canceled になる。理由はログに出す)。
		slog.ErrorContext(ctx, "write query rejected in transaction", slog.String("reason", err.Error()))
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		return t.conn.QueryRowContext(canceled, query, args...)
	}
	return t.conn.QueryRowContext(ctx, query, args...)
}

func (t *immediateTx) Commit() error {
	if t.aborted != nil {
		return t.aborted
	}
	_, err := t.conn.ExecContext(context.Background(), "COMMIT")
	return err
}

func (t *immediateTx) Rollback() error {
	if t.aborted != nil {
		return nil // DB が既に取り消している
	}
	_, err := t.conn.ExecContext(context.Background(), "ROLLBACK")
	return err
}

// runWriteTx は begin(BEGIN IMMEDIATE か BEGIN CONCURRENT)で1回分のトランザクションを実行する。
// 失敗時は必ず ROLLBACK し、ROLLBACK 自体に失敗した接続はプールへ戻さず破棄する
// (トランザクションを握ったままの接続がプールに戻ると、以後の書き込みが全てロックされるため)。
func runWriteTx(db *gorm.DB, begin string, fn func(tx *gorm.DB) error) (err error) {
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

	itx := &immediateTx{conn: conn, concurrent: begin == "BEGIN CONCURRENT"}
	tx := db.Session(&gorm.Session{Context: ctx, NewDB: true})
	tx.Statement.ConnPool = itx

	if err = tx.Exec(begin).Error; err != nil {
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
