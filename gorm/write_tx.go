package gw_gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"gorm.io/gorm"
)

// WriteTxOptions は WriteTransactionWithOptions の設定。
// MaxAttempts・BaseBackoff・MaxBackoff がゼロ値なら、UseWriteRetry の設定(呼んでいなければ DefaultWriteTxOptions)の
// 値を使う。負の値はエラーになる。
type WriteTxOptions struct {
	// Immediate を true にすると、UseWriteRetry の Concurrent が有効な DB でも、このトランザクションは BEGIN IMMEDIATE で始める。
	// 使う場面: DDL(CREATE TABLE、ALTER TABLE など。BEGIN CONCURRENT の中では使えない)を流すとき。
	// 同じ行に書き込みが集中する処理(連番の採番など)で、衝突してやり直しを繰り返すより順番に待つほうが速いとき。
	Immediate bool
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

func (o WriteTxOptions) withDefaults() WriteTxOptions { return o.fillFrom(DefaultWriteTxOptions()) }

// fillFrom は、ゼロ値のやり直しの回数・待ち時間を、fallback の値で埋める。
// fallback は UseWriteRetry で決めた設定か、DefaultWriteTxOptions。
func (o WriteTxOptions) fillFrom(fallback WriteTxOptions) WriteTxOptions {
	if o.MaxAttempts == 0 {
		o.MaxAttempts = fallback.MaxAttempts
	}
	if o.BaseBackoff == 0 {
		o.BaseBackoff = fallback.BaseBackoff
	}
	if o.MaxBackoff == 0 {
		o.MaxBackoff = fallback.MaxBackoff
	}
	return o
}

// validate は負の値を拒む(呼び出し側の誤り。黙って既定値にすると誤りが隠れ、MaxAttempts が負だと fn が一度も実行されない)。
func (o WriteTxOptions) validate() error {
	if o.MaxAttempts < 0 || o.BaseBackoff < 0 || o.MaxBackoff < 0 {
		return fmt.Errorf("gw_gorm: やり直しの回数と待ち時間に負の値は使えない: %+v", o)
	}
	return nil
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
//   - DB の外への副作用(通知の送信など)を持たせない
//   - 書き込む値(モデルなど)は fn の中で作る。DB がトランザクションを取り消しても Go の値は戻らない
//     (fn の外で作ったモデルには、取り消された INSERT の ID などが残り、やり直しで別の SQL になる)
//   - fn の外の変数(集計や戻り値)には、WriteTransaction が成功した後に結果を入れる
//
// UseWriteRetry を呼んだ DB では、そこで指定したやり直しの設定を使う。
func WriteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return WriteTransactionWithOptions(db, WriteTxOptions{}, fn)
}

// WriteTransactionWithOptions は WriteTransaction の設定(Immediate、やり直しの回数と待ち時間)を指定できる版。
func WriteTransactionWithOptions(db *gorm.DB, opts WriteTxOptions, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	if db.Error != nil {
		return db.Error
	}
	if err := opts.validate(); err != nil {
		return err
	}
	if !IsSQLiteDialect(db) || isInsideTransaction(db) {
		return db.Transaction(fn)
	}
	begin := "BEGIN IMMEDIATE"
	if p := writeRetryPluginOf(db); p != nil {
		opts = opts.fillFrom(p.retry)
		if p.concurrent && !opts.Immediate {
			begin = "BEGIN CONCURRENT"
		}
	} else {
		opts = opts.withDefaults()
	}
	return retryWrite(statementContext(db), opts, func() error {
		return runWriteTx(db, begin, fn)
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

// retryWrite は、fn がやり直せるエラー(isRetryableWriteError)で失敗した場合に限り、待ち時間を少しずつ延ばしながらやり直す。
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
	// broken は、DB がトランザクションを終わらせた、または終わらせたかもしれないエラー(endsTransaction で判定する)。
	//
	// 保険: DB が終わらせた後の SQL は、トランザクションの外で 1 文ずつその場で確定してしまう。fn がエラーを返さずに
	// 続けた場合(内側の Transaction のエラーを捨てた、など)に一部だけが残らないよう、以後の SQL と COMMIT を止める。
	// ROLLBACK は必ず流す(トランザクションが続いている場合もあるため)。
	// shortcut: Raw(...).Scan / Rows / Row のエラーは行を読む途中で返り、この接続の層からは見えない。そのエラーを捨てて
	// 続けた場合は守れない(fn はエラーを返すこと)。
	broken error
}

// markBrokenIfEnded は、err がトランザクションを終わらせる(ことがある)エラー(endsTransaction)なら、broken に記録する。
func (t *immediateTx) markBrokenIfEnded(err error) {
	if t.broken == nil && endsTransaction(err) {
		t.broken = err
	}
}

// endsTransaction は、DB がトランザクションを終わらせる(ことがある)エラーかを返す。
// Turso の MVCC の衝突(Write-write conflict)は必ず終わらせる。SQLite のディスク一杯・I/O エラー・メモリ不足・中断は
// 終わらせることも、その文だけで済むこともある(https://www.sqlite.org/lang_transaction.html#response_to_errors_within_a_transaction)。
func endsTransaction(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{"write-write conflict", "database or disk is full", "disk i/o error", "out of memory", "interrupted"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

var (
	_ gorm.ConnPool    = (*immediateTx)(nil)
	_ gorm.TxCommitter = (*immediateTx)(nil)
)

func (t *immediateTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if t.broken != nil {
		return nil, t.broken
	}
	return t.conn.PrepareContext(ctx, query)
}

func (t *immediateTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if t.broken != nil {
		return nil, t.broken
	}
	res, err := t.conn.ExecContext(ctx, query, args...)
	t.markBrokenIfEnded(err)
	return res, err
}

func (t *immediateTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if t.broken != nil {
		return nil, t.broken
	}
	rows, err := t.conn.QueryContext(ctx, query, args...)
	t.markBrokenIfEnded(err)
	return rows, err
}

func (t *immediateTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if t.broken != nil {
		// *sql.Row にエラーを持たせる手段が無いので、キャンセル済みの context で呼んで SQL を実行させない。
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		return t.conn.QueryRowContext(canceled, query, args...)
	}
	return t.conn.QueryRowContext(ctx, query, args...)
}

func (t *immediateTx) Commit() error {
	if t.broken != nil {
		return t.broken
	}
	_, err := t.conn.ExecContext(context.Background(), "COMMIT")
	t.markBrokenIfEnded(err) // 衝突は COMMIT の時点で返ることもある
	return err
}

func (t *immediateTx) Rollback() error {
	_, err := t.conn.ExecContext(context.Background(), "ROLLBACK")
	if err != nil && t.broken != nil && containsFold(err, "no transaction is active") {
		return nil // DB が既に終わらせていた。接続にトランザクションは残っていない
	}
	return err
}

// runWriteTx は、begin(BEGIN IMMEDIATE か BEGIN CONCURRENT)で 1 回分のトランザクションを実行する。
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

	itx := &immediateTx{conn: conn}
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
