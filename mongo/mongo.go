package gw_mongo

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"time"
	"unicode"

	gw_arrays "github.com/generalworksinc/goutil/arrays"
	gw_errors "github.com/generalworksinc/goutil/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultConnectTimeout    = 30 * time.Second
	defaultOperationTimeout  = 30 * time.Second
	defaultDisconnectTimeout = 10 * time.Second
	defaultDialTimeout       = 30 * time.Second
	defaultDialKeepAlive     = 300 * time.Second
	defaultMaxConnIdleTime   = 10 * time.Second
)

type Config struct {
	URI               string
	Database          string
	ConnectTimeout    time.Duration
	OperationTimeout  time.Duration
	DisconnectTimeout time.Duration
	DialTimeout       time.Duration
	DialKeepAlive     time.Duration
	MaxConnIdleTime   time.Duration
}

type Client struct {
	client            *mongo.Client
	database          *mongo.Database
	operationTimeout  time.Duration
	disconnectTimeout time.Duration
}

type Session struct {
	session           mongo.Session
	operationTimeout  time.Duration
	disconnectTimeout time.Duration
}

type Database[T Model] struct {
	database         *mongo.Database
	session          *Session
	idValue          interface{} // 条件filterに `_id` を追加するための値
	operationTimeout time.Duration
}

func NewClient(conf Config) (*Client, error) {
	if strings.TrimSpace(conf.URI) == "" {
		return nil, gw_errors.New("mongo uri is empty")
	}
	if strings.TrimSpace(conf.Database) == "" {
		return nil, gw_errors.New("mongo database is empty")
	}

	normalizeConfig(&conf)

	clientOptions := options.Client().ApplyURI(conf.URI)
	dialer := &net.Dialer{
		Timeout:   conf.DialTimeout,
		KeepAlive: conf.DialKeepAlive,
	}
	clientOptions.SetMaxConnIdleTime(conf.MaxConnIdleTime)
	clientOptions.SetDialer(dialer)

	ctx, cancel := context.WithTimeout(context.Background(), conf.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	return &Client{
		client:            client,
		database:          client.Database(conf.Database),
		operationTimeout:  conf.OperationTimeout,
		disconnectTimeout: conf.DisconnectTimeout,
	}, nil
}

func normalizeConfig(conf *Config) {
	if conf.ConnectTimeout <= 0 {
		conf.ConnectTimeout = defaultConnectTimeout
	}
	if conf.OperationTimeout <= 0 {
		conf.OperationTimeout = defaultOperationTimeout
	}
	if conf.DisconnectTimeout <= 0 {
		conf.DisconnectTimeout = defaultDisconnectTimeout
	}
	if conf.DialTimeout <= 0 {
		conf.DialTimeout = defaultDialTimeout
	}
	if conf.DialKeepAlive <= 0 {
		conf.DialKeepAlive = defaultDialKeepAlive
	}
	if conf.MaxConnIdleTime <= 0 {
		conf.MaxConnIdleTime = defaultMaxConnIdleTime
	}
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.disconnectTimeout)
	defer cancel()
	return gw_errors.Wrap(c.client.Disconnect(ctx))
}

func NewDatabase[T Model](c *Client) *Database[T] {
	return &Database[T]{
		database:         c.database,
		operationTimeout: c.operationTimeout,
	}
}

func (c *Client) StartSession(txOpts ...*options.TransactionOptions) (*Session, error) {
	if c == nil || c.client == nil {
		return nil, gw_errors.New("mongo client is not initialized")
	}
	session, err := c.client.StartSession()
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	if err := session.StartTransaction(txOpts...); err != nil {
		session.EndSession(context.Background())
		return nil, gw_errors.Wrap(err)
	}
	return &Session{
		session:           session,
		operationTimeout:  c.operationTimeout,
		disconnectTimeout: c.disconnectTimeout,
	}, nil
}

func (s *Session) Close() {
	if s == nil || s.session == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.disconnectTimeout)
	defer cancel()
	s.session.EndSession(ctx)
}

func (s *Session) Commit() error {
	if s == nil || s.session == nil {
		return gw_errors.New("mongo session is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.operationTimeout)
	defer cancel()
	return gw_errors.Wrap(s.session.CommitTransaction(ctx))
}

func (s *Session) Abort() error {
	if s == nil || s.session == nil {
		return gw_errors.New("mongo session is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.operationTimeout)
	defer cancel()
	return gw_errors.Wrap(s.session.AbortTransaction(ctx))
}

func (db *Database[T]) Session(session *Session) *Database[T] {
	db.session = session
	return db
}

func (db *Database[T]) Id(id string) *Database[T] {
	if modelUsesStringID[T]() {
		db.idValue = id
		return db
	}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		db.idValue = oid
	} else {
		db.idValue = id
	}
	return db
}

func (db *Database[T]) Oid(oid *primitive.ObjectID) *Database[T] {
	if oid == nil {
		db.idValue = nil
		return db
	}
	db.idValue = *oid
	return db
}

func (db *Database[T]) DeleteOne(filter interface{}, opts ...*options.DeleteOptions) (int64, error) {
	collection := db.database.Collection(collectionName[T]())
	condFilter, err := db.cond(filter)
	if err != nil {
		return 0, gw_errors.Wrap(err)
	}
	ctx, cancel := db.operationContext()
	defer cancel()
	result, err := collection.DeleteOne(ctx, condFilter, opts...)
	if err != nil {
		return 0, gw_errors.Wrap(err)
	}
	return result.DeletedCount, nil
}

func (db *Database[T]) UpdateOne(filter interface{}, update interface{}, opts ...*options.UpdateOptions) (int64, error) {
	updateVal, err := db.buildUpdateValue(update)
	if err != nil {
		return 0, gw_errors.Wrap(err)
	}
	collection := db.database.Collection(collectionName[T]())
	condFilter, err := db.cond(filter)
	if err != nil {
		return 0, gw_errors.Wrap(err)
	}
	ctx, cancel := db.operationContext()
	defer cancel()
	result, err := collection.UpdateOne(ctx, condFilter, bson.D{{"$set", updateVal}}, opts...)
	if err != nil {
		return 0, gw_errors.Wrap(err)
	}
	return result.ModifiedCount, nil
}

// 更新時はupdateDocumentをもとに、もしupdateDocumentがnilなら、entityをまるごと更新する
// 更新対象のカラムが指定されている場合
// →指定したカラムに加えて、updatedAtを強制的に更新する
// 更新対象のカラムが指定されていない場合
// → 全カラムが対象、ただし値が初期値（Zero）のカラムは、更新対象にしない
func (db *Database[T]) UpsertOne(entity *T, filter interface{}, updateFields []string, opts ...*options.FindOneAndUpdateOptions) (*T, error) {
	collection := db.database.Collection(collectionName[T]())
	idValue, err := applyBaseModelFields(entity, nil, false, true)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	if !idIsZero(idValue) {
		db.idValue = idValue
	}

	if filter == nil && idIsZero(idValue) {
		// データが存在しないので、insert処理を行う
		return db.InsertOne(entity)
	}

	entityParams, err := marshalToPrimitiveD(entity)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	var updateParams primitive.D
	if len(updateFields) > 0 {
		// 更新対象のカラムが指定されている場合
		// →指定したカラムに加えて、updatedAtを強制的に更新する
		for _, elem := range entityParams {
			if elem.Key == "updatedAt" || gw_arrays.ContainsString(updateFields, elem.Key) {
				updateParams = append(updateParams, elem)
			}
		}
	} else {
		// 更新対象のカラムが指定されていない場合
		// → 全カラムが対象、ただし値が初期値（Zero）のカラムは、更新対象にしない
		for _, elem := range entityParams {
			if elem.Key != "createdAt" && elem.Key != "_id" && !isZero(elem.Value) {
				updateParams = append(updateParams, elem)
			}
		}
	}

	condFilter, err := db.cond(filter)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	ctx, cancel := db.operationContext()
	defer cancel()
	result := collection.FindOneAndUpdate(ctx, condFilter, bson.D{{"$set", updateParams}}, opts...)
	if err := result.Err(); err == nil {
		return entity, nil
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		// データが存在しないので、insert処理を行う
		if _, updateErr := applyBaseModelFields(entity, nil, true, true); updateErr != nil {
			return nil, gw_errors.Wrap(updateErr)
		}
		return db.InsertOne(entity)
	} else {
		return nil, gw_errors.Wrap(err)
	}
}

func (db *Database[T]) FindOne(filter interface{}, opts ...*options.FindOneOptions) (*T, error) {
	collection := db.database.Collection(collectionName[T]())
	condFilter, err := db.cond(filter)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	ctx, cancel := db.operationContext()
	defer cancel()
	result := collection.FindOne(ctx, condFilter, opts...)
	if err := result.Err(); err != nil {
		//データが存在しない場合は、エラーではなくnilを返す
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, gw_errors.Wrap(err)
	}
	var resultStruct T
	if err := result.Decode(&resultStruct); err != nil {
		return nil, gw_errors.Wrap(err)
	}
	return &resultStruct, nil
}

func (db *Database[T]) Find(filter interface{}, opts ...*options.FindOptions) ([]*T, error) {
	collection := db.database.Collection(collectionName[T]())
	condFilter, err := db.cond(filter)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	ctx, cancel := db.operationContext()
	defer cancel()
	cur, err := collection.Find(ctx, condFilter, opts...)
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, nil
		}
		return nil, gw_errors.Wrap(err)
	}
	defer cur.Close(ctx)

	resultList := []*T{}
	if err := cur.All(ctx, &resultList); err != nil {
		return nil, gw_errors.Wrap(err)
	}
	return resultList, nil
}

// 登録されたIDを、entityにセットして返す
func (db *Database[T]) InsertOne(entity *T, opts ...*options.InsertOneOptions) (*T, error) {
	collection := db.database.Collection(collectionName[T]())
	if _, err := applyBaseModelFields(entity, nil, true, true); err != nil {
		return nil, gw_errors.Wrap(err)
	}
	ctx, cancel := db.operationContext()
	defer cancel()
	result, err := collection.InsertOne(ctx, entity, opts...)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	if _, err := applyBaseModelFields(entity, result.InsertedID, false, false); err != nil {
		return nil, gw_errors.Wrap(err)
	}
	return entity, nil
}

func (db *Database[T]) operationContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), db.operationTimeout)
	if db.session == nil {
		return ctx, cancel
	}
	return mongo.NewSessionContext(ctx, db.session.session), cancel
}

func (db *Database[T]) cond(filter interface{}) (interface{}, error) {
	if db.idValue == nil {
		if filter == nil {
			return bson.D{}, nil
		}
		return filter, nil
	}
	if filter == nil {
		return bson.D{{"_id", db.idValue}}, nil
	}

	switch f := filter.(type) {
	case primitive.E:
		return bson.D{{"_id", db.idValue}, f}, nil
	case primitive.A:
		return bson.D{{"_id", db.idValue}, {"$and", f}}, nil
	case primitive.M:
		return bson.D{{"_id", db.idValue}, {"$and", bson.A{f}}}, nil
	case primitive.D:
		return bson.D{{"_id", db.idValue}, {"$and", bson.A{f}}}, nil
	default:
		return nil, gw_errors.Errorf("filter type is not primitive.E / primitive.A / primitive.M / primitive.D: %T", filter)
	}
}

func (db *Database[T]) buildUpdateValue(update interface{}) (interface{}, error) {
	now := time.Now()

	if bsonD, ok := update.(primitive.D); ok {
		// "_id", "created_at"は更新対象から除外
		updateParams := primitive.D{{"updatedAt", now}}
		for _, elem := range bsonD {
			if elem.Key != "createdAt" && elem.Key != "_id" {
				updateParams = append(updateParams, elem)
			}
		}
		return updateParams, nil
	}

	if updateMap, ok := update.(map[string]interface{}); ok {
		// "_id", "created_at"は更新対象から除外
		updateParams := map[string]interface{}{}
		for k, v := range updateMap {
			if k != "createdAt" && k != "_id" {
				updateParams[k] = v
			}
		}
		updateParams["updatedAt"] = now
		return updateParams, nil
	}

	if bsonM, ok := update.(primitive.M); ok {
		// "_id", "created_at"は更新対象から除外
		updateParams := bson.M{}
		for k, v := range bsonM {
			if k != "createdAt" && k != "_id" {
				updateParams[k] = v
			}
		}
		updateParams["updatedAt"] = now
		return updateParams, nil
	}

	if t, ok := update.(*T); ok {
		// "_id", "createdAt"は更新対象から除外
		if _, err := applyBaseModelFields(t, nil, false, true); err != nil {
			return nil, gw_errors.Wrap(err)
		}
		entityParams, err := marshalToPrimitiveD(t)
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
		updateParams := primitive.D{}
		for _, elem := range entityParams {
			if elem.Key != "createdAt" && elem.Key != "_id" {
				updateParams = append(updateParams, elem)
			}
		}
		return updateParams, nil
	}

	return nil, gw_errors.Errorf("unsupported update type: %T", update)
}

func marshalToPrimitiveD(v interface{}) (primitive.D, error) {
	bsonBytes, err := bson.Marshal(v)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	var params primitive.D
	if err := bson.Unmarshal(bsonBytes, &params); err != nil {
		return nil, gw_errors.Wrap(err)
	}
	return params, nil
}

func collectionName[T Model]() string {
	return toSnakeCase((*new(T)).StructName())
}

// スネークケースに変換する関数
func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if unicode.IsUpper(r) {
			if i != 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func applyBaseModelFields[T any](obj *T, updateID interface{}, updateCreateTime bool, updateUpdateTime bool) (interface{}, error) {
	if obj == nil {
		return nil, gw_errors.New("obj is nil")
	}
	structValue := reflect.ValueOf(obj).Elem()
	fieldVal, err := findBaseModelField(structValue)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	idField := fieldVal.FieldByName("ID")
	if !idField.IsValid() {
		return nil, gw_errors.New("base model has no ID field")
	}

	if updateID != nil {
		if err := setIDValue(idField, updateID); err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}

	now := time.Now()
	if updateCreateTime {
		if err := setTimeValue(fieldVal, "CreatedAt", now); err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}
	if updateUpdateTime {
		if err := setTimeValue(fieldVal, "UpdatedAt", now); err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}

	return idField.Interface(), nil
}

func findBaseModelField(structValue reflect.Value) (reflect.Value, error) {
	candidates := baseModelFieldCandidates()
	for _, name := range candidates {
		field := structValue.FieldByName(name)
		if field.IsValid() {
			return field, nil
		}
	}
	return reflect.Value{}, gw_errors.New("embedded mongo base model field not found")
}

func baseModelFieldCandidates() []string {
	return []string{
		"MongoBaseModel", "BaseModel",
		"MongoBaseModelSimple", "BaseModelSimple",
		"MongoBaseModelUuid", "MongoBaseModelUUID", "BaseModelUuid", "BaseModelUUID",
		"MongoBaseModelUuidSimple", "MongoBaseModelUUIDSimple", "BaseModelUuidSimple", "BaseModelUUIDSimple",
	}
}

func modelUsesStringID[T Model]() bool {
	modelType := reflect.TypeOf((*T)(nil)).Elem()
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	if modelType.Kind() != reflect.Struct {
		return false
	}
	candidates := baseModelFieldCandidates()
	for _, name := range candidates {
		if field, ok := modelType.FieldByName(name); ok {
			if idField, ok := field.Type.FieldByName("ID"); ok {
				return idField.Type.Kind() == reflect.String
			}
		}
	}
	return false
}

func setTimeValue(baseField reflect.Value, fieldName string, t time.Time) error {
	field := baseField.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}
	if !field.CanSet() {
		return gw_errors.New("cannot set time field: " + fieldName)
	}
	if field.Type() != reflect.TypeOf(time.Time{}) {
		return gw_errors.New("time field is not time.Time: " + fieldName)
	}
	field.Set(reflect.ValueOf(t))
	return nil
}

func setIDValue(idField reflect.Value, updateID interface{}) error {
	objectIDType := reflect.TypeOf(primitive.ObjectID{})
	switch {
	case idField.Type() == objectIDType:
		switch v := updateID.(type) {
		case primitive.ObjectID:
			idField.Set(reflect.ValueOf(v))
		case *primitive.ObjectID:
			if v == nil {
				idField.Set(reflect.ValueOf(primitive.NilObjectID))
			} else {
				idField.Set(reflect.ValueOf(*v))
			}
		case string:
			if strings.TrimSpace(v) == "" {
				idField.Set(reflect.ValueOf(primitive.NilObjectID))
			} else {
				oid, err := primitive.ObjectIDFromHex(v)
				if err != nil {
					return gw_errors.Wrap(err)
				}
				idField.Set(reflect.ValueOf(oid))
			}
		default:
			return gw_errors.Errorf("unsupported object id type: %T", updateID)
		}
		return nil
	case idField.Kind() == reflect.String:
		switch v := updateID.(type) {
		case string:
			idField.SetString(v)
		case primitive.ObjectID:
			idField.SetString(v.Hex())
		case *primitive.ObjectID:
			if v == nil {
				idField.SetString("")
			} else {
				idField.SetString(v.Hex())
			}
		default:
			return gw_errors.Errorf("unsupported string id type: %T", updateID)
		}
		return nil
	default:
		value := reflect.ValueOf(updateID)
		if value.IsValid() && value.Type().AssignableTo(idField.Type()) {
			idField.Set(value)
			return nil
		}
		return gw_errors.Errorf("unsupported id field type=%s update id type=%T", idField.Type().String(), updateID)
	}
}

func idIsZero(id interface{}) bool {
	if id == nil {
		return true
	}
	switch v := id.(type) {
	case primitive.ObjectID:
		return v.IsZero()
	case *primitive.ObjectID:
		return v == nil || v.IsZero()
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return isZero(v)
	}
}

func isZero(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return isZeroValue(v)
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		//ここでゼロ値チェックするかは実装次第
		//ここでは全フィールドがゼロ値なら該当structもゼロ値とみなす
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Invalid:
		return true
	default:
		return false
	}
}
