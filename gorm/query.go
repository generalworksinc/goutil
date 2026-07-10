package gw_gorm

import "gorm.io/gorm"

// FindOne は条件に一致する最大1件を検索する。
//
// GORM の First は 0 件を ErrRecordNotFound（合成エラー）にするが、FindOne は
// 「不在は正常系」として (nil, nil) を返す。エラーが返るのは接続断・制約違反・
// テナントスコープ未設定など、予期しない異常だけになる。
//
//	user, err := gw_gorm.FindOne[models.User](db.Where("email = ?", email))
//	if err != nil { ... }      // 予期せぬエラーのみ
//	if user == nil { ... }     // 不在（エラーではない）
//
// 使い分け: 不在があり得る検索は FindOne、存在しなければバグという検索は First を使う。
func FindOne[T any](db *gorm.DB, conds ...interface{}) (*T, error) {
	var ent T
	result := db.Limit(1).Find(&ent, conds...)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &ent, nil
}
