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
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	Id string `sql:"type:varchar(46)" json:"id"`
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
type BaseModelByManualIdLogicalDelUlid struct {
	CreatedAt time.Time  `xorm:"datetime(3) created" json:"createdAt"`
	UpdatedAt time.Time  `xorm:"updated" json:"updatedAt"`
	DeletedAt *time.Time `xorm:"deleted" json:"deletedAt"`
}

func (entity *BaseModelByManualIdLogicalDelUlid) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

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
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	Id string `sql:"type:varchar(46)" json:"id"`
}

func (entity *BaseModelSimpleUlid) BeforeInsert() {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
}
