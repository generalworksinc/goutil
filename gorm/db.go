package gw_gorm

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DefaultConfig は GORM の標準設定を返す。
// - テーブル名は単数形 snake_case（SingularTable。struct 名がそのままテーブル名になるため TableName() の実装は不要）
// - FK 制約を DDL に含めない（データを pure に保つ方針。リレーションはアプリ層のタグ定義で扱う）
// - タイムスタンプは UTC
// - debug 時のみ SQL ログを出力
// DB ドライバ（Dialector）と DSN の組み立てはアプリ側の責務。goutil は特定の DB に依存しない。
// テナントガード（UseTenantGuard）も用途ごとのライフサイクルに合わせて呼び出し側で登録する。
func DefaultConfig(debug bool) *gorm.Config {
	config := &gorm.Config{
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
	if debug {
		config.Logger = logger.Default.LogMode(logger.Info)
	}
	return config
}

// ReCreateTable は開発・デモ用の破壊的なテーブル再作成ユーティリティ。
func ReCreateTable(db *gorm.DB, models ...any) error {
	if err := db.Migrator().DropTable(models...); err != nil {
		return err
	}
	return db.AutoMigrate(models...)
}
