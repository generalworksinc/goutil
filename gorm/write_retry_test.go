package gw_gorm

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	_ "turso.tech/database/tursogo"
)

type retryRow struct {
	ID     uint `gorm:"primaryKey"`
	N      int
	Status string `gorm:"default:active"` // 既定値のある列があると、GORM は Create で RETURNING を使う
}

// openTursoMVCC は Turso Database(tursogo)のファイル DB を MVCC にして開き、UseWriteRetry を有効にする。
func openTursoMVCC(t *testing.T, concurrent bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.New(sqlite.Config{DriverName: "turso", DSN: filepath.Join(t.TempDir(), "mvcc.db")}), DefaultConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = sqlDB.Close() })
	var mode string
	if err := db.Raw("PRAGMA journal_mode = 'mvcc'").Scan(&mode).Error; err != nil || mode != "mvcc" {
		t.Fatalf("journal_mode = %q, %v", mode, err)
	}
	if err := db.AutoMigrate(&retryRow{}); err != nil {
		t.Fatal(err)
	}
	if err := UseWriteRetry(db, WriteRetryOptions{Concurrent: concurrent, Retry: WriteTxOptions{MaxAttempts: 100, BaseBackoff: time.Millisecond, MaxBackoff: 20 * time.Millisecond}}); err != nil {
		t.Fatal(err)
	}
	return db
}

// holdConcurrentUpdate は別の接続で BEGIN CONCURRENT のまま行を更新し、d の後にコミットする。
func holdConcurrentUpdate(t *testing.T, db *gorm.DB, id uint, n int, d time.Duration) <-chan error {
	t.Helper()
	sqlDB, _ := db.DB()
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, "BEGIN CONCURRENT"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "UPDATE retry_row SET n = ? WHERE id = ?", n, id); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		time.Sleep(d)
		_, err := conn.ExecContext(ctx, "COMMIT")
		conn.Close()
		done <- err
	}()
	return done
}

// 読んでから書くトランザクションを並行させても、更新を失わない(衝突したものはやり直される)。
func TestUseWriteRetryConcurrentReadModifyWriteLosesNoUpdates(t *testing.T) {
	db := openTursoMVCC(t, true)
	if p := writeRetryPluginOf(db); p == nil || !p.concurrent {
		t.Fatal("MVCC の DB では BEGIN CONCURRENT を使うはず")
	}
	row := retryRow{N: 0}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	const workers, each = 8, 10
	var wg sync.WaitGroup
	errs := make(chan error, workers*each)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				errs <- WriteTransaction(db, func(tx *gorm.DB) error {
					var r retryRow
					if err := tx.First(&r, row.ID).Error; err != nil {
						return err
					}
					return tx.Model(&r).Update("n", r.N+1).Error
				})
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var got retryRow
	db.First(&got, row.ID)
	if got.N != workers*each {
		t.Fatalf("n = %d, want %d", got.N, workers*each)
	}
}

// トランザクションの外の 1 文の書き込みも、BEGIN CONCURRENT の途中の行とぶつかればやり直される。
func TestUseWriteRetryRetriesAutocommitWrites(t *testing.T) {
	db := openTursoMVCC(t, true)
	row := retryRow{N: 0}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	committed := holdConcurrentUpdate(t, db, row.ID, 1, 150*time.Millisecond)
	if err := db.Model(&retryRow{}).Where("id = ?", row.ID).Update("n", 2).Error; err != nil {
		t.Fatalf("UPDATE: %v", err)
	}
	if err := <-committed; err != nil {
		t.Fatal(err)
	}

	// RETURNING 付き(衝突のエラーは行を読む途中で返る)
	committed = holdConcurrentUpdate(t, db, row.ID, 3, 150*time.Millisecond)
	var returned []retryRow
	if err := db.Model(&returned).Clauses(clause.Returning{}).Where("id = ?", row.ID).Update("n", 4).Error; err != nil {
		t.Fatalf("UPDATE RETURNING: %v", err)
	}
	if err := <-committed; err != nil {
		t.Fatal(err)
	}
	if len(returned) != 1 || returned[0].N != 4 {
		t.Fatalf("returned = %+v", returned)
	}

	committed = holdConcurrentUpdate(t, db, row.ID, 5, 150*time.Millisecond)
	if err := db.Exec("UPDATE retry_row SET n = ? WHERE id = ?", 6, row.ID).Error; err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if err := <-committed; err != nil {
		t.Fatal(err)
	}
	var got retryRow
	db.First(&got, row.ID)
	if got.N != 6 {
		t.Fatalf("n = %d, want 6(後から書いた側が、相手のコミットの後に通る)", got.N)
	}
}

// DDL は BEGIN CONCURRENT の中で使えないので、BEGIN IMMEDIATE に切り替えてやり直す。
func TestWriteTransactionFallsBackToImmediateForDDL(t *testing.T) {
	db := openTursoMVCC(t, true)
	type ddlRow struct{ ID uint }
	if err := WriteTransaction(db, func(tx *gorm.DB) error {
		return tx.Migrator().CreateTable(&ddlRow{})
	}); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable(&ddlRow{}) {
		t.Fatal("table was not created")
	}
}

func TestWriteTransactionNestedInConcurrent(t *testing.T) {
	db := openTursoMVCC(t, true)
	inner := errors.New("inner")
	if err := WriteTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&retryRow{N: 1}).Error; err != nil {
			return err
		}
		_ = tx.Transaction(func(tx2 *gorm.DB) error {
			if err := tx2.Create(&retryRow{N: 2}).Error; err != nil {
				return err
			}
			return inner // SAVEPOINT まで戻す
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&retryRow{}).Count(&n)
	if n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}

// MVCC でない DB では Concurrent を指定しても BEGIN IMMEDIATE のまま動く。
func TestUseWriteRetryIgnoresConcurrentWithoutMVCC(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "plain.db"))
	if err := UseWriteRetry(db, WriteRetryOptions{Concurrent: true}); err != nil {
		t.Fatal(err)
	}
	if p := writeRetryPluginOf(db); p == nil || p.concurrent {
		t.Fatal("MVCC でない DB で BEGIN CONCURRENT を使ってはいけない")
	}
	if err := WriteTransaction(db, func(tx *gorm.DB) error { return tx.Create(&writeTxRow{V: "a"}).Error }); err != nil {
		t.Fatal(err)
	}
}

func TestIsRetryableWriteError(t *testing.T) {
	for msg, want := range map[string]bool{
		"database is locked (5) (SQLITE_BUSY)":                     true,
		"failed to execute SQL:\nSQLite error: database is locked": true,
		"turso: database is busy: database is locked":              true,
		"turso: error: Write-write conflict":                       true,
		"turso: Tursodb error: Write-write conflict":               true,
		"turso: Transaction timed out":                             true,
		"UNIQUE constraint failed: retry_row.id":                   false,
		"Transaction error: DDL statements require an exclusive transaction (use BEGIN instead of BEGIN CONCURRENT)": false,
	} {
		if got := IsRetryableWriteError(errors.New(msg)); got != want {
			t.Errorf("IsRetryableWriteError(%q) = %v, want %v", msg, got, want)
		}
	}
	if !needsExclusiveTx(errors.New("Transaction error: DDL statements require an exclusive transaction")) {
		t.Error("needsExclusiveTx")
	}
}

// 内側の Transaction で衝突し、そのエラーを外側が握りつぶして書き込みを続けても、一部だけが残らない。
func TestNestedConflictSwallowedByOuterDoesNotPersistPartially(t *testing.T) {
	db := openTursoMVCC(t, true)
	row := retryRow{N: 0}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	attempts := 0
	fn := func(tx *gorm.DB) error {
		attempts++
		_ = tx.Transaction(func(tx2 *gorm.DB) error {
			return tx2.Model(&retryRow{}).Where("id = ?", row.ID).Update("n", 9).Error
		}) // 衝突のエラーを握りつぶす
		return tx.Create(&retryRow{N: 100}).Error
	}

	// やり直さない設定: 衝突で失敗し、後続の Create も残らない
	committed := holdConcurrentUpdate(t, db, row.ID, 1, 300*time.Millisecond)
	err := WriteTransactionWithOptions(db, WriteTxOptions{MaxAttempts: 1}, fn)
	if !isWriteConflict(err) {
		t.Fatalf("err = %v, want Write-write conflict", err)
	}
	if err := <-committed; err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&retryRow{}).Where("n = 100").Count(&n)
	if n != 0 {
		t.Fatalf("取り消されたトランザクションの後続の書き込みが %d 件残った", n)
	}

	// やり直す設定: 相手のコミットの後に全体がやり直され、Create はちょうど 1 件
	attempts = 0
	committed = holdConcurrentUpdate(t, db, row.ID, 1, 100*time.Millisecond)
	if err := WriteTransaction(db, fn); err != nil {
		t.Fatal(err)
	}
	if err := <-committed; err != nil {
		t.Fatal(err)
	}
	db.Model(&retryRow{}).Where("n = 100").Count(&n)
	if n != 1 || attempts < 2 {
		t.Fatalf("rows = %d, attempts = %d", n, attempts)
	}
}

// 結果の行を読んだ後に失敗した 1 文の書き込みは、流し直さない(Dest の対応が崩れるため)。
func TestRetryingSkipsAfterRowsWereScanned(t *testing.T) {
	db := openTursoMVCC(t, false)
	p := &writeRetryPlugin{retry: WriteTxOptions{MaxAttempts: 5, BaseBackoff: time.Millisecond, MaxBackoff: time.Millisecond}}
	for _, scanned := range []int64{0, 1} {
		calls := 0
		stmt := db.Session(&gorm.Session{NewDB: true})
		p.retrying(func(d *gorm.DB) {
			calls++
			d.Error = errors.New("turso: error: Write-write conflict")
			d.RowsAffected = scanned
		})(stmt)
		want := 5
		if scanned > 0 {
			want = 1
		}
		if calls != want {
			t.Errorf("scanned=%d: calls = %d, want %d", scanned, calls, want)
		}
	}
}

// UseWriteRetry のやり直しの設定は WriteTransaction にも効く。
func TestWriteTransactionUsesUseWriteRetryOptions(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "opts.db"))
	if err := UseWriteRetry(db, WriteRetryOptions{Retry: WriteTxOptions{MaxAttempts: 3, BaseBackoff: time.Millisecond, MaxBackoff: time.Millisecond}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	_ = WriteTransaction(db, func(*gorm.DB) error {
		calls++
		return errors.New("database is locked")
	})
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}
