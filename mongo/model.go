package gw_mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model interface {
	StructName() string
}

type BaseModel struct {
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	// CreatedAt   string `bson:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type BaseModelSimple struct {
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`
}

type BaseModelUUID struct {
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	ID string `json:"id" bson:"_id,omitempty"`
	// CreatedAt   string `bson:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type BaseModelUUIDSimple struct {
	//idは自動的にprimary keyに設定される（※UUIDをセット）
	ID string `json:"id" bson:"_id,omitempty"`
}
