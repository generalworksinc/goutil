package gw_gorm

import (
	"fmt"
	"time"

	gw_uuid "github.com/generalworksinc/goutil/uuid"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	// idは自動的にprimary keyに設定される（※UUIDをセット）
	Id        string    `gorm:"type:varchar(46);primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (entity *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if entity.Id == "" {
		entity.Id = uuid.New().String()
	}
	return nil
}

type BaseModelLogicalDel struct {
	// idは自動的にprimary keyに設定される（※UUIDをセット）
	Id        string         `gorm:"type:varchar(46);primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (entity *BaseModelLogicalDel) BeforeCreate(tx *gorm.DB) error {
	if entity.Id == "" {
		entity.Id = uuid.New().String()
	}
	return nil
}

func (entity *BaseModelLogicalDel) GetIdStr() string {
	return fmt.Sprintf("%v", entity.Id)
}

func (entity *BaseModelLogicalDel) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelByManualId struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (entity *BaseModelByManualId) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelByManualIdLogicalDel struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (entity *BaseModelByManualIdLogicalDel) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelSimple struct {
	// idは自動的にprimary keyに設定される（※UUIDをセット）
	Id string `gorm:"type:varchar(46);primaryKey" json:"id"`
}

func (entity *BaseModelSimple) BeforeCreate(tx *gorm.DB) error {
	if entity.Id == "" {
		entity.Id = uuid.New().String()
	}
	return nil
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type BaseModelUlid struct {
	// idは自動的にprimary keyに設定される（※ULIDをセット）
	Id        string    `gorm:"type:varchar(46);primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (entity *BaseModelUlid) BeforeCreate(tx *gorm.DB) error {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
	return nil
}

type BaseModelLogicalDelUlid struct {
	// idは自動的にprimary keyに設定される（※ULIDをセット）
	Id        string         `gorm:"type:varchar(46);primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (entity *BaseModelLogicalDelUlid) BeforeCreate(tx *gorm.DB) error {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
	return nil
}

func (entity *BaseModelLogicalDelUlid) GetIdStr() string {
	return fmt.Sprintf("%v", entity.Id)
}

func (entity *BaseModelLogicalDelUlid) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelByManualIdUlid struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (entity *BaseModelByManualIdUlid) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelByManualIdLogicalDelUlid struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (entity *BaseModelByManualIdLogicalDelUlid) IsCreated() bool {
	return entity.CreatedAt == time.Time{}
}

type BaseModelSimpleUlid struct {
	// idは自動的にprimary keyに設定される（※ULIDをセット）
	Id string `gorm:"type:varchar(46);primaryKey" json:"id"`
}

func (entity *BaseModelSimpleUlid) BeforeCreate(tx *gorm.DB) error {
	if entity.Id == "" {
		entity.Id = gw_uuid.GetUlid()
	}
	return nil
}
