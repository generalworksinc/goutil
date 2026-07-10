package gw_authz

import (
	"errors"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

// CasbinRule はポリシーの永続化テーブル。マイグレーションで ReCreateTable / AutoMigrate に渡す。
// SingularTable 命名でテーブル名は casbin_rule になる。
//
// 公式の casbin gorm-adapter はパッケージレベルで全DBドライバ（mysql/postgres/...）を import するため
// 「goutil は特定の DB に依存しない」原則に反する。このファイルは persist.Adapter を
// gorm.io/gorm だけで実装した薄い代替（今回使う範囲で機能同等）。
type CasbinRule struct {
	Id    uint   `gorm:"primaryKey;autoIncrement"`
	Ptype string `gorm:"type:varchar(16);index"`
	V0    string `gorm:"type:varchar(256)"`
	V1    string `gorm:"type:varchar(256)"`
	V2    string `gorm:"type:varchar(256)"`
	V3    string `gorm:"type:varchar(256)"`
	V4    string `gorm:"type:varchar(256)"`
	V5    string `gorm:"type:varchar(256)"`
}

type gormAdapter struct {
	db *gorm.DB
}

var _ persist.Adapter = (*gormAdapter)(nil)

func newGormAdapter(db *gorm.DB) *gormAdapter {
	return &gormAdapter{db: db}
}

func (r *CasbinRule) toLine() string {
	parts := []string{r.Ptype}
	for _, v := range []string{r.V0, r.V1, r.V2, r.V3, r.V4, r.V5} {
		if v == "" {
			break
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, ", ")
}

func ruleFrom(ptype string, rule []string) CasbinRule {
	ent := CasbinRule{Ptype: ptype}
	fields := []*string{&ent.V0, &ent.V1, &ent.V2, &ent.V3, &ent.V4, &ent.V5}
	for i, v := range rule {
		if i >= len(fields) {
			break
		}
		*fields[i] = v
	}
	return ent
}

// LoadPolicy は全ルールを読み込んで model へ反映する。
func (a *gormAdapter) LoadPolicy(m model.Model) error {
	var rules []CasbinRule
	if err := a.db.Order("id ASC").Find(&rules).Error; err != nil {
		return err
	}
	for i := range rules {
		if err := persist.LoadPolicyLine(rules[i].toLine(), m); err != nil {
			return err
		}
	}
	return nil
}

// SavePolicy は model の全ルールでテーブルを置き換える。
func (a *gormAdapter) SavePolicy(m model.Model) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&CasbinRule{}).Error; err != nil {
			return err
		}
		for _, sec := range []string{"p", "g"} {
			for ptype, ast := range m[sec] {
				for _, rule := range ast.Policy {
					ent := ruleFrom(ptype, rule)
					if err := tx.Create(&ent).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// AddPolicy は1ルールを追加する（enforcer の autoSave 経由で呼ばれる）。
func (a *gormAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	ent := ruleFrom(ptype, rule)
	return a.db.Create(&ent).Error
}

// RemovePolicy は一致する1ルールを削除する。
func (a *gormAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	ent := ruleFrom(ptype, rule)
	return a.db.Where(&ent, "Ptype", "V0", "V1", "V2", "V3", "V4", "V5").Delete(&CasbinRule{}).Error
}

// RemoveFilteredPolicy は fieldIndex 位置から fieldValues に一致するルールを削除する。
func (a *gormAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	if fieldIndex < 0 || fieldIndex+len(fieldValues) > 6 {
		return errors.New("gw_authz: invalid fieldIndex/fieldValues for RemoveFilteredPolicy")
	}
	query := a.db.Where("ptype = ?", ptype)
	columns := []string{"v0", "v1", "v2", "v3", "v4", "v5"}
	for i, v := range fieldValues {
		if v == "" {
			continue // casbin の慣例: 空はワイルドカード
		}
		query = query.Where(columns[fieldIndex+i]+" = ?", v)
	}
	return query.Delete(&CasbinRule{}).Error
}
