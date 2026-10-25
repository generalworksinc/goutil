package gw_gorm

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// WriteRetryOptions は UseWriteRetry の設定。
type WriteRetryOptions struct {
	// Concurrent を true にすると、DB が Turso Database の MVCC(PRAGMA journal_mode = mvcc)のとき、
	// WriteTransaction を BEGIN CONCURRENT で始め、別々の行への書き込みを並行させる。
	// それ以外の DB(SQLite、libSQL など)では無視され、BEGIN IMMEDIATE のまま動く。
	Concurrent bool
	// Retry はやり直しの回数と待ち時間。WriteTransaction と 1 文の書き込みの両方に効く。ゼロ値は DefaultWriteTxOptions。
	Retry WriteTxOptions
}

const writeRetryPluginName = "gw_gorm:write_retry"

// UseWriteRetry は DB を開いた直後に一度だけ呼ぶ。以後、利用側は書き込みの衝突を意識しなくてよい。
//
//   - WriteTransaction: BEGIN CONCURRENT(Concurrent が有効なとき)か BEGIN IMMEDIATE で始め、衝突・ロック待ち・
//     タイムアウトで失敗したらトランザクション全体をやり直す。DDL は BEGIN CONCURRENT の中で使えないため、
//     その失敗は BEGIN IMMEDIATE に切り替えてやり直す
//   - トランザクションの外の 1 文の書き込み(Create / Update / Delete / Exec): 同じ理由で失敗したら、その操作を流し直す
//
// やり直して安全な理由: 衝突・ロック待ちで失敗した書き込みは DB に何も残さない(トランザクションは全体が取り消され、
// 1 文の書き込みは反映されない)。ぶつかった相手の更新は DB が検知して止めるので、上書きもしない。
// 守ること: WriteTransaction の fn に DB の外への副作用(通知の送信など)を持たせず、書き込む値は fn の中で作る
// (WriteTransaction のコメントを参照)。
//
// GORM の既定のトランザクション(Create などを 1 回ずつ BEGIN〜COMMIT で包む)は、素の BEGIN で始まりやり直せないため
// 切る(SkipDefaultTransaction)。複数の SQL になる書き込みは WriteTransaction で包むこと。
// SQLite 系以外の DB では何もしない。
func UseWriteRetry(db *gorm.DB, opts WriteRetryOptions) error {
	if !IsSQLiteDialect(db) {
		return nil
	}
	p := &writeRetryPlugin{retry: opts.Retry.withDefaults()}
	if opts.Concurrent {
		var mode string
		if err := db.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
			return err
		}
		p.concurrent = strings.EqualFold(mode, "mvcc")
	}
	db.Config.SkipDefaultTransaction = true
	return db.Use(p)
}

type writeRetryPlugin struct {
	concurrent bool
	retry      WriteTxOptions
}

func (p *writeRetryPlugin) Name() string { return writeRetryPluginName }

func (p *writeRetryPlugin) Initialize(db *gorm.DB) error {
	callbacks := db.Callback()
	for _, c := range []struct {
		processor interface {
			Get(string) func(*gorm.DB)
			Replace(string, func(*gorm.DB)) error
		}
		name string
	}{
		{callbacks.Create(), "gorm:create"},
		{callbacks.Update(), "gorm:update"},
		{callbacks.Delete(), "gorm:delete"},
		{callbacks.Raw(), "gorm:raw"},
	} {
		next := c.processor.Get(c.name)
		if next == nil {
			continue
		}
		if err := c.processor.Replace(c.name, p.retrying(next)); err != nil {
			return err
		}
	}
	return nil
}

// retrying はトランザクションの外の 1 文の書き込みを、やり直せるエラーのときに流し直す。
// RETURNING 付きの書き込みは、衝突のエラーが行を読む途中で返るため、接続の層ではなくこの単位で流し直す。
// トランザクションの中ではやり直さない(Turso は衝突でトランザクション全体を取り消すため、外側の WriteTransaction がやり直す)。
//
// 結果の行を 1 行でも読んだ後に失敗した場合(RowsAffected > 0)はやり直さない。GORM は読んだ行を Dest に書き込んでおり
// (ID を埋めた要素は次の回で飛ばす、など)、そのまま流し直すと結果の対応が崩れるため。Turso の衝突は最初の行の前に返る。
func (p *writeRetryPlugin) retrying(next func(*gorm.DB)) func(*gorm.DB) {
	return func(db *gorm.DB) {
		if isInsideTransaction(db) {
			itx, ok := db.Statement.ConnPool.(*immediateTx)
			if !ok {
				next(db)
				return
			}
			itx.covered++
			next(db)
			itx.covered--
			itx.markAborted(db.Error)
			return
		}
		ctx := statementContext(db)
		for attempt := 1; ; attempt++ {
			next(db)
			if db.Error == nil || !isRetryableWriteError(db.Error) || db.RowsAffected > 0 || attempt == p.retry.MaxAttempts {
				return
			}
			wait := p.retry.backoff(attempt)
			slog.WarnContext(ctx, "write hit a retryable error; retrying",
				slog.Int("attempt", attempt), slog.Duration("wait", wait), slog.String("error", db.Error.Error()))
			if !sleepCtx(ctx, wait) {
				return
			}
			db.Error = nil
			db.RowsAffected = 0
		}
	}
}

func writeRetryPluginOf(db *gorm.DB) *writeRetryPlugin {
	p, _ := db.Config.Plugins[writeRetryPluginName].(*writeRetryPlugin)
	return p
}

// isRetryableWriteError は、やり直せば通りうる書き込みのエラーかを返す。
// ロック待ち(database is locked / busy)、Turso の MVCC の衝突(Write-write conflict)、
// libSQL のサーバーが他の書き込みを待たせすぎたトランザクションを打ち切ったとき(Transaction timed out)。
func isRetryableWriteError(err error) bool {
	return IsSQLiteBusyError(err) || isWriteConflict(err) || containsFold(err, "transaction timed out")
}

func isWriteConflict(err error) bool { return containsFold(err, "write-write conflict") }

// needsExclusiveTx は、BEGIN CONCURRENT の中で DDL を流したときのエラーかを返す。
func needsExclusiveTx(err error) bool { return containsFold(err, "require an exclusive transaction") }

func containsFold(err error, s string) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), s)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
