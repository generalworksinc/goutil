package gw_models

import (
	"fmt"
	gw_uuid "github.com/generalworksinc/goutil/uuid"
	"github.com/google/uuid"
	"time"
)

type BaseModel struct {
	//idは自動的にprimary keyに設定される（※UUIdをセット）
	Id        string    `xorm:"varchar(46) pk" json:"id"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated" json:"updatedAt"`
}

func (entity *BaseModel) BeforeInsert() {
	if entity.Id == "" {
		uuid := uuid.New()
		entity.Id = uuid.String()
	}
}

type BaseModelLogicalDel struct {
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	Id        string     `xorm:"varchar(46) pk" json:"id"`
	CreatedAt time.Time  `xorm:"created" json:"createdAt"`
	UpdatedAt time.Time  `xorm:"updated" json:"updatedAt"`
	DeletedAt *time.Time `xorm:"deleted" json:"deletedAt"`
}
// 【注意】ByManualId 系の base は Id フィールドを持たない。
// 埋め込む側の struct が必ず自前で pk タグ付きの Id を宣言すること。
//
//	Id string `xorm:"varchar(46) pk" json:"id"`
//
// pk を忘れるとコンパイルも Insert も通るが、そのモデルへの .ID() 呼び出し
// (Get/Update/Delete)が実行時に「ID condition is error, expect 0 primarykeys」
// エラーで全て失敗する(RefreshToken の失効処理が壊れていた実例あり)。
type BaseModelByManualIdLogicalDel struct {
	CreatedAt time.Time  `xorm:"created" json:"createdAt"`
	UpdatedAt time.Time  `xorm:"updated" json:"updatedAt"`
	DeletedAt *time.Time `xorm:"deleted" json:"deletedAt"`
}

func (entity *BaseModelByManualIdLogicalDel) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}
func (entity *BaseModelLogicalDel) BeforeInsert() {
	//log.Println("call beforeInsert!!", entity.Id == "", strings.TrimSpace(entity.Id) == "")
	if entity.Id == "" {
		uuid := uuid.New()
		entity.Id = uuid.String()
	}
}
func (entity *BaseModelLogicalDel) GetIdStr() string {
	fmt.Println("entity:", entity)
	return fmt.Sprintf("%v", entity.Id)
}
func (entity *BaseModelLogicalDel) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelSimple struct {
	//idはprimary key（※UUIDをセット）
	//※旧タグは `sql:"type:varchar(46)"` で、xorm が読まない無効なタグだったため
	//  primary key が付かないテーブルが生成されていた。既にこの base を埋め込んで
	//  運用中のテーブルがある場合、バージョンアップ後の Sync2 で PK が付与される。
	Id string `xorm:"varchar(46) pk" json:"id"`
}

func (entity *BaseModelSimple) BeforeInsert() {
	if entity.Id == "" {
		uuid := uuid.New()
		entity.Id = uuid.String()
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type BaseModelUlid struct {
	//idは自動的にprimary keyに設定される（※UUIdをセット）
	Id        string    `xorm:"varchar(46) pk" json:"id"`
	CreatedAt time.Time `xorm:"datetime(3) created" json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated" json:"updatedAt"`
}

func (entity *BaseModelUlid) BeforeInsert() {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
}

type BaseModelLogicalDelUlid struct {
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	Id        string     `xorm:"varchar(46) pk" json:"id"`
	CreatedAt time.Time  `xorm:"datetime(3) created" json:"createdAt"`
	UpdatedAt time.Time  `xorm:"updated" json:"updatedAt"`
	DeletedAt *time.Time `xorm:"deleted" json:"deletedAt"`
}
// 【注意】Id フィールドを持たない。埋め込む側が必ず pk タグ付きの Id を自前宣言すること
// (詳細は BaseModelByManualIdLogicalDel のコメント参照)。
type BaseModelByManualIdLogicalDelUlid struct {
	CreatedAt time.Time  `xorm:"datetime(3) created" json:"createdAt"`
	UpdatedAt time.Time  `xorm:"updated" json:"updatedAt"`
	DeletedAt *time.Time `xorm:"deleted" json:"deletedAt"`
}

func (entity *BaseModelByManualIdLogicalDelUlid) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

// 【注意】Id フィールドを持たない。埋め込む側が必ず pk タグ付きの Id を自前宣言すること
// (詳細は BaseModelByManualIdLogicalDel のコメント参照)。
type BaseModelByManualId struct {
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated" json:"updatedAt"`
}

func (entity *BaseModelLogicalDelUlid) BeforeInsert() {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
}
func (entity *BaseModelLogicalDelUlid) GetIdStr() string {
	return fmt.Sprintf("%v", entity.Id)
}
func (entity *BaseModelLogicalDelUlid) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelSimpleUlid struct {
	//idはprimary key（※ULIDをセット）
	//※旧タグは `sql:"type:varchar(46)"` で、xorm が読まない無効なタグだった(BaseModelSimple と同様)。
	Id string `xorm:"varchar(46) pk" json:"id"`
}

func (entity *BaseModelSimpleUlid) BeforeInsert() {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
}
