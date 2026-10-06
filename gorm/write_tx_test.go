package gw_gorm

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type writeTxRow struct {
	ID uint `gorm:"primaryKey"`
	V  string
}

// openWriteTxDB はファイル SQLite を MaxOpenConns(1) で開く(アプリの運用と同じ構成)。
// Windows では開いたままの SQLite ファイルを TempDir が削除できないため、必ず閉じる。
func openWriteTxDB(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), DefaultConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&writeTxRow{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func countWriteTxRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&writeTxRow{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWriteTransactionCommitsOnSuccessAndRollsBackOnError(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "commit.db"))

	if err := WriteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&writeTxRow{V: "committed"}).Error
	}); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("boom")
	err := WriteTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&writeTxRow{V: "rolled back"}).Error; err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected callback error to propagate, got %v", err)
	}
	if n := countWriteTxRows(t, db); n != 1 {
		t.Fatalf("expected exactly the committed row, got %d", n)
	}
}

func TestWriteTransactionRollsBackOnPanic(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "panic.db"))

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("panic must propagate")
			}
		}()
		_ = WriteTransaction(db, func(tx *gorm.DB) error {
			if err := tx.Create(&writeTxRow{V: "panicked"}).Error; err != nil {
				return err
			}
			panic("boom")
		})
	}()
	if n := countWriteTxRows(t, db); n != 0 {
		t.Fatalf("row must be rolled back after panic, got %d rows", n)
	}
	// 接続がトランザクションを握ったままプールへ戻っていないこと(次の書き込みが通る)。
	if err := WriteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&writeTxRow{V: "after panic"}).Error
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteTransactionWorksWithGormCreateSaveDelete(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "callbacks.db"))

	row := &writeTxRow{V: "created"}
	err := WriteTransaction(db, func(tx *gorm.DB) error {
		// Create/Save/Delete は GORM が既定トランザクション(gorm:begin_transaction)を張ろうとするが、
		// 既にトランザクション中の接続なのでスキップされなければならない。
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		row.V = "saved"
		if err := tx.Save(row).Error; err != nil {
			return err
		}
		other := &writeTxRow{V: "deleted"}
		if err := tx.Create(other).Error; err != nil {
			return err
		}
		return tx.Delete(other).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	var rows []writeTxRow
	if err := db.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].V != "saved" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestWriteTransactionSupportsNestedTransactions(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "nested.db"))

	err := WriteTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&writeTxRow{V: "outer"}).Error; err != nil {
			return err
		}
		// Repository 層が db.Transaction を使う場合は SAVEPOINT によるネストになること。
		if err := tx.Transaction(func(inner *gorm.DB) error {
			return inner.Create(&writeTxRow{V: "inner"}).Error
		}); err != nil {
			return err
		}
		// 既にトランザクション中の db に WriteTransaction を重ねても BEGIN IMMEDIATE を重ねず、ネストとして扱うこと。
		if err := WriteTransaction(tx, func(inner *gorm.DB) error {
			return inner.Create(&writeTxRow{V: "inner write tx"}).Error
		}); err != nil {
			return err
		}
		// ネストしたトランザクションのエラーは SAVEPOINT までロールバックされ、外側は続行できること。
		nestedErr := tx.Transaction(func(inner *gorm.DB) error {
			if err := inner.Create(&writeTxRow{V: "inner rolled back"}).Error; err != nil {
				return err
			}
			return errors.New("nested failure")
		})
		if nestedErr == nil {
			t.Fatal("nested transaction error must propagate")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := countWriteTxRows(t, db); n != 3 {
		t.Fatalf("expected outer+inner+inner write tx rows, got %d", n)
	}
}

func TestWriteTransactionWaitsForAnotherWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.db")
	db := openWriteTxDB(t, path)
	other := openWriteTxDB(t, path)

	lockHeld := make(chan struct{})
	released := make(chan error, 1)
	go func() {
		released <- other.Connection(func(conn *gorm.DB) error {
			if err := conn.Exec("BEGIN IMMEDIATE").Error; err != nil {
				close(lockHeld)
				return err
			}
			close(lockHeld)
			time.Sleep(150 * time.Millisecond)
			return conn.Exec("COMMIT").Error
		})
	}()
	<-lockHeld

	start := time.Now()
	if err := WriteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&writeTxRow{V: "after lock"}).Error
	}); err != nil {
		t.Fatalf("WriteTransaction should succeed once the other writer releases the lock: %v", err)
	}
	if err := <-released; err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("expected to wait for the other writer, finished in %v", elapsed)
	}
	if n := countWriteTxRows(t, db); n != 1 {
		t.Fatalf("expected one row, got %d", n)
	}
}

func TestWriteTransactionUsesStatementContext(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "ctx.db"))
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "request")

	if err := WriteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		if got := tx.Statement.Context.Value(key{}); got != "request" {
			t.Fatalf("transaction must inherit the request context, got %v", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRetryOnSQLiteBusyRetriesOnlyBusyErrors(t *testing.T) {
	opts := WriteTxOptions{MaxAttempts: 4, BaseBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}
	busy := errors.New("failed to execute SQL:\nSQLite error: database is locked")

	calls := 0
	err := retryOnSQLiteBusy(context.Background(), opts, func() error {
		calls++
		if calls < 3 {
			return busy
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("expected success on third attempt, err=%v calls=%d", err, calls)
	}

	calls = 0
	other := errors.New("UNIQUE constraint failed")
	if err := retryOnSQLiteBusy(context.Background(), opts, func() error {
		calls++
		return other
	}); !errors.Is(err, other) || calls != 1 {
		t.Fatalf("non-busy errors must not be retried, err=%v calls=%d", err, calls)
	}

	calls = 0
	if err := retryOnSQLiteBusy(context.Background(), opts, func() error {
		calls++
		return busy
	}); !errors.Is(err, busy) || calls != opts.MaxAttempts {
		t.Fatalf("expected give up after %d attempts, err=%v calls=%d", opts.MaxAttempts, err, calls)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	if err := retryOnSQLiteBusy(cancelled, opts, func() error {
		calls++
		return busy
	}); !errors.Is(err, busy) || calls != 1 {
		t.Fatalf("cancelled context must stop retrying, err=%v calls=%d", err, calls)
	}
}

func TestWriteTxOptionsDefaultsAndBackoff(t *testing.T) {
	opts := WriteTxOptions{}.withDefaults()
	if opts != DefaultWriteTxOptions() {
		t.Fatalf("zero options must fall back to defaults, got %+v", opts)
	}
	custom := WriteTxOptions{MaxAttempts: 2, BaseBackoff: 10 * time.Millisecond, MaxBackoff: 15 * time.Millisecond}
	for attempt, max := range map[int]time.Duration{1: 20 * time.Millisecond, 2: 30 * time.Millisecond, 5: 30 * time.Millisecond} {
		wait := custom.backoff(attempt)
		if wait < 10*time.Millisecond || wait > max {
			t.Errorf("attempt %d: wait %v out of range", attempt, wait)
		}
	}
}

func TestIsSQLiteBusyError(t *testing.T) {
	for _, msg := range []string{
		"database is locked",
		"database is locked (5) (SQLITE_BUSY)",
		"failed to execute SQL:\nSQLite error: database is locked",
		"database table is locked",
	} {
		if !IsSQLiteBusyError(errors.New(msg)) {
			t.Errorf("%q should be a busy error", msg)
		}
	}
	if IsSQLiteBusyError(errors.New("UNIQUE constraint failed")) || IsSQLiteBusyError(nil) {
		t.Error("non-busy errors must not match")
	}
}

func TestIsSQLiteDialect(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "dialect.db"))
	if !IsSQLiteDialect(db) {
		t.Fatal("sqlite driver must be detected")
	}
	dummy, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if IsSQLiteDialect(dummy) || IsSQLiteDialect(nil) {
		t.Fatal("non-sqlite dialects must not be detected")
	}
}
