package gw_mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type unitObjectModel struct {
	BaseModel `bson:",inline"`
	Name      string `bson:"name"`
}

func (unitObjectModel) StructName() string {
	return "unitObjectModel"
}

type unitStringModel struct {
	BaseModelUUID `bson:",inline"`
	Name          string `bson:"name"`
}

func (unitStringModel) StructName() string {
	return "unitStringModel"
}

func TestToSnakeCase(t *testing.T) {
	if got := toSnakeCase("refreshToken"); got != "refresh_token" {
		t.Fatalf("unexpected snake case. got=%s", got)
	}
	if got := toSnakeCase("LetsEncrypt"); got != "lets_encrypt" {
		t.Fatalf("unexpected snake case. got=%s", got)
	}
}

func TestModelUsesStringID(t *testing.T) {
	if modelUsesStringID[unitObjectModel]() {
		t.Fatalf("object model should not use string id")
	}
	if !modelUsesStringID[unitStringModel]() {
		t.Fatalf("string model should use string id")
	}
}

func TestIdSelectionByModelType(t *testing.T) {
	hexID := primitive.NewObjectID().Hex()

	objDB := &Database[unitObjectModel]{}
	objDB.Id(hexID)
	if _, ok := objDB.idValue.(primitive.ObjectID); !ok {
		t.Fatalf("object model should keep ObjectID for hex input. actual=%T", objDB.idValue)
	}

	objDB.Id("not-hex")
	if got, ok := objDB.idValue.(string); !ok || got != "not-hex" {
		t.Fatalf("object model should keep string for non-hex. value=%v type=%T", objDB.idValue, objDB.idValue)
	}

	strDB := &Database[unitStringModel]{}
	strDB.Id(hexID)
	if got, ok := strDB.idValue.(string); !ok || got != hexID {
		t.Fatalf("string model should keep string as-is. value=%v type=%T", strDB.idValue, strDB.idValue)
	}
}

func TestCond(t *testing.T) {
	db := &Database[unitObjectModel]{}

	cond, err := db.cond(nil)
	if err != nil {
		t.Fatalf("cond error: %v", err)
	}
	if d, ok := cond.(bson.D); !ok || len(d) != 0 {
		t.Fatalf("expected empty bson.D, got=%T %v", cond, cond)
	}

	oid := primitive.NewObjectID()
	db.idValue = oid
	cond, err = db.cond(nil)
	if err != nil {
		t.Fatalf("cond error: %v", err)
	}
	d, ok := cond.(bson.D)
	if !ok || len(d) != 1 || d[0].Key != "_id" || d[0].Value != oid {
		t.Fatalf("unexpected cond with id only: %#v", cond)
	}

	cond, err = db.cond(primitive.M{"name": "alice"})
	if err != nil {
		t.Fatalf("cond error: %v", err)
	}
	d, ok = cond.(bson.D)
	if !ok || len(d) != 2 || d[1].Key != "$and" {
		t.Fatalf("unexpected cond with filter: %#v", cond)
	}

	if _, err := db.cond(123); err == nil {
		t.Fatalf("unsupported filter type should return error")
	}
}

func TestBuildUpdateValue(t *testing.T) {
	db := &Database[unitObjectModel]{}

	objID := primitive.NewObjectID()
	dVal, err := db.buildUpdateValue(primitive.D{{Key: "_id", Value: objID}, {Key: "createdAt", Value: time.Now()}, {Key: "name", Value: "alice"}})
	if err != nil {
		t.Fatalf("buildUpdateValue(D) error: %v", err)
	}
	dUpdate, ok := dVal.(primitive.D)
	if !ok {
		t.Fatalf("expected primitive.D, got %T", dVal)
	}
	dMap := dToMap(dUpdate)
	if _, exists := dMap["_id"]; exists {
		t.Fatalf("_id should be excluded: %#v", dUpdate)
	}
	if _, exists := dMap["createdAt"]; exists {
		t.Fatalf("createdAt should be excluded: %#v", dUpdate)
	}
	if _, exists := dMap["updatedAt"]; !exists {
		t.Fatalf("updatedAt should be injected: %#v", dUpdate)
	}
	if got := dMap["name"]; got != "alice" {
		t.Fatalf("name mismatch. got=%v", got)
	}

	mVal, err := db.buildUpdateValue(map[string]interface{}{"_id": objID, "createdAt": time.Now(), "name": "bob"})
	if err != nil {
		t.Fatalf("buildUpdateValue(map) error: %v", err)
	}
	mUpdate, ok := mVal.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", mVal)
	}
	if _, exists := mUpdate["_id"]; exists {
		t.Fatalf("_id should be excluded: %#v", mUpdate)
	}
	if _, exists := mUpdate["createdAt"]; exists {
		t.Fatalf("createdAt should be excluded: %#v", mUpdate)
	}
	if _, exists := mUpdate["updatedAt"]; !exists {
		t.Fatalf("updatedAt should be injected: %#v", mUpdate)
	}

	entity := &unitObjectModel{Name: "charlie"}
	eVal, err := db.buildUpdateValue(entity)
	if err != nil {
		t.Fatalf("buildUpdateValue(entity) error: %v", err)
	}
	eUpdate, ok := eVal.(primitive.D)
	if !ok {
		t.Fatalf("expected primitive.D, got %T", eVal)
	}
	eMap := dToMap(eUpdate)
	if _, exists := eMap["_id"]; exists {
		t.Fatalf("_id should be excluded: %#v", eUpdate)
	}
	if _, exists := eMap["createdAt"]; exists {
		t.Fatalf("createdAt should be excluded: %#v", eUpdate)
	}
	if _, exists := eMap["updatedAt"]; !exists {
		t.Fatalf("updatedAt should be injected: %#v", eUpdate)
	}
	if got := eMap["name"]; got != "charlie" {
		t.Fatalf("name mismatch. got=%v", got)
	}
}

func TestApplyBaseModelFields(t *testing.T) {
	obj := &unitObjectModel{}
	idValue, err := applyBaseModelFields(obj, nil, true, true)
	if err != nil {
		t.Fatalf("applyBaseModelFields object error: %v", err)
	}
	if _, ok := idValue.(primitive.ObjectID); !ok {
		t.Fatalf("expected ObjectID id type, got %T", idValue)
	}
	if obj.CreatedAt.IsZero() || obj.UpdatedAt.IsZero() {
		t.Fatalf("timestamps should be set. createdAt=%v updatedAt=%v", obj.CreatedAt, obj.UpdatedAt)
	}

	oid := primitive.NewObjectID()
	_, err = applyBaseModelFields(obj, oid, false, false)
	if err != nil {
		t.Fatalf("applyBaseModelFields object id set error: %v", err)
	}
	if obj.ID != oid {
		t.Fatalf("object id mismatch. expected=%s got=%s", oid.Hex(), obj.ID.Hex())
	}

	str := &unitStringModel{}
	_, err = applyBaseModelFields(str, oid, true, true)
	if err != nil {
		t.Fatalf("applyBaseModelFields string model error: %v", err)
	}
	if str.ID != oid.Hex() {
		t.Fatalf("string model id mismatch. expected=%s got=%s", oid.Hex(), str.ID)
	}
	if str.CreatedAt.IsZero() || str.UpdatedAt.IsZero() {
		t.Fatalf("timestamps should be set for string model")
	}

	_, err = applyBaseModelFields(obj, "invalid-object-id", false, false)
	if err == nil {
		t.Fatalf("invalid object id string should return error")
	}
}

func dToMap(d primitive.D) map[string]interface{} {
	m := make(map[string]interface{}, len(d))
	for _, e := range d {
		m[e.Key] = e.Value
	}
	return m
}
