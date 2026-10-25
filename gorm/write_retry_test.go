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

// holdConcurrentUpdate は、別の接続で BEGIN CONCURRENT のまま行を更新し、d だけ待ってからコミットする。
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

// トランザクションの外の 1 文の書き込みも、BEGIN CONCURRENT の途中の行と衝突すれば、やり直される。
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

// DDL は BEGIN CONCURRENT の中で使えない。Immediate を指定したトランザクションなら流せる。
func TestDDLNeedsImmediate(t *testing.T) {
	db := openTursoMVCC(t, true)
	type ddlRow struct{ ID uint }
	createTable := func(tx *gorm.DB) error { return tx.Migrator().CreateTable(&ddlRow{}) }
	if err := WriteTransaction(db, createTable); err == nil {
		t.Fatal("BEGIN CONCURRENT の中の DDL はエラーになるはず")
	}
	if err := WriteTransactionWithOptions(db, WriteTxOptions{Immediate: true}, createTable); err != nil {
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

func TestRetryableWriteErrorClassification(t *testing.T) {
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
		if got := isRetryableWriteError(errors.New(msg)); got != want {
			t.Errorf("isRetryableWriteError(%q) = %v, want %v", msg, got, want)
		}
	}
}

// 内側の Transaction で衝突し、そのエラーを外側が捨てて書き込みを続けても、一部だけが残ることはない。
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

func TestMarkBrokenIfEnded(t *testing.T) {
	for msg, want := range map[string]bool{
		"turso: error: Write-write conflict":     true,
		"database or disk is full":               true,
		"disk I/O error":                         true,
		"out of memory":                          true,
		"interrupted":                            true,
		"database is locked":                     false, // BEGIN IMMEDIATE で待つ側のエラー。トランザクションは続く
		"UNIQUE constraint failed: retry_row.id": false, // その文だけが失敗し、トランザクションは続く
	} {
		tx := &immediateTx{}
		tx.markBrokenIfEnded(errors.New(msg))
		if got := tx.broken != nil; got != want {
			t.Errorf("markBrokenIfEnded(%q): broken = %v, want %v", msg, got, want)
		}
	}
}

// トランザクションを終わらせる「ことがある」エラーでは、後続の SQL を止め、ROLLBACK は実際に流す
// (トランザクションが続いていた場合に、終わっていない接続をプールへ返さないため)。
func TestBrokenTxStillRollsBackWhenTransactionSurvives(t *testing.T) {
	db := openWriteTxDB(t, filepath.Join(t.TempDir(), "survive.db"))
	err := WriteTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&writeTxRow{V: "kept?"}).Error; err != nil {
			return err
		}
		itx := tx.Statement.ConnPool.(*immediateTx)
		itx.markBrokenIfEnded(errors.New("database or disk is full")) // SQLite はこの文だけ取り消してトランザクションを続けることがある
		if err := tx.Create(&writeTxRow{V: "after"}).Error; err == nil {
			t.Error("壊れたトランザクションの後続の SQL は止めるはず")
		}
		return nil // エラーを捨てて終わる
	})
	if err == nil {
		t.Fatal("COMMIT は壊れたトランザクションのエラーで失敗するはず")
	}
	// ROLLBACK が流れていれば、同じ接続(MaxOpenConns=1)で次の書き込みができ、最初の Create も残っていない
	if err := WriteTransaction(db, func(tx *gorm.DB) error { return tx.Create(&writeTxRow{V: "next"}).Error }); err != nil {
		t.Fatalf("接続にトランザクションが残っている: %v", err)
	}
	if n := countWriteTxRows(t, db); n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}

// Immediate を指定したトランザクション同士は、同じ行でも衝突せずに順番を待つ(やり直しを許さなくても失敗しない)。
// 同じ処理を BEGIN CONCURRENT で行うと、やり直しを許さなければ衝突で失敗する。
func TestImmediateOptionSerializesHotRow(t *testing.T) {
	db := openTursoMVCC(t, true)
	run := func(immediate bool) (n int, failures int) {
		row := retryRow{N: 0}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		var mu sync.Mutex
		var wg sync.WaitGroup
		for w := 0; w < 8; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					err := WriteTransactionWithOptions(db, WriteTxOptions{Immediate: immediate, MaxAttempts: 1}, func(tx *gorm.DB) error {
						var r retryRow
						if err := tx.First(&r, row.ID).Error; err != nil {
							return err
						}
						return tx.Model(&r).Update("n", r.N+1).Error
					})
					if err != nil {
						mu.Lock()
						failures++
						mu.Unlock()
					}
				}
			}()
		}
		wg.Wait()
		var got retryRow
		db.First(&got, row.ID)
		return got.N, failures
	}
	if n, failures := run(true); n != 80 || failures != 0 {
		t.Fatalf("Immediate: n = %d, failures = %d, want 80, 0", n, failures)
	}
	if _, failures := run(false); failures == 0 {
		t.Log("BEGIN CONCURRENT でも衝突が起きなかった(負荷が足りない)。Immediate の側の結果だけを判定に使う")
	}
}

type busyHookRow struct {
	ID uint `gorm:"primaryKey"`
	V  string
}

func (busyHookRow) BeforeCreate(*gorm.DB) error { return errors.New("database is locked") }

// 前段のフックのやり直せるエラーを、書き込みの失敗と取り違えない(フックを飛ばして書き込みだけを流さない)。
func TestRetryingDoesNotSwallowHookErrors(t *testing.T) {
	db := openTursoMVCC(t, false)
	if err := db.AutoMigrate(&busyHookRow{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&busyHookRow{V: "x"}).Error; err == nil {
		t.Fatal("BeforeCreate のエラーが返るはず")
	}
	var n int64
	db.Model(&busyHookRow{}).Count(&n)
	if n != 0 {
		t.Fatalf("rows = %d, want 0", n)
	}
}
