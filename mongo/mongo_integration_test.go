package gw_mongo

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type itObjectModel struct {
	BaseModel `bson:",inline"`
	Name      string `bson:"name"`
}

func (itObjectModel) StructName() string {
	return "goutilMongoItObject"
}

type itStringModel struct {
	BaseModelUUID `bson:",inline"`
	Name          string `bson:"name"`
}

func (itStringModel) StructName() string {
	return "goutilMongoItString"
}

var (
	tcOnce     sync.Once
	tcErr      error
	tcMongoURI string
	tcMongoDB  string
	tcMongoCtr testcontainers.Container
)

func TestMain(m *testing.M) {
	code := m.Run()
	if tcMongoCtr != nil {
		_ = tcMongoCtr.Terminate(context.Background())
	}
	os.Exit(code)
}

func TestMongoIntegration_ObjectIDCRUD(t *testing.T) {
	client := mustIntegrationClient(t)
	db := NewDatabase[itObjectModel](client)

	entity := &itObjectModel{Name: "alice"}
	inserted, err := db.InsertOne(entity)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if inserted.ID.IsZero() {
		t.Fatalf("inserted object id should not be zero")
	}

	t.Cleanup(func() {
		_, _ = db.Id(inserted.ID.Hex()).DeleteOne(nil)
	})

	found, err := db.Id(inserted.ID.Hex()).FindOne(nil)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found == nil || found.Name != "alice" {
		t.Fatalf("unexpected find result: %#v", found)
	}

	_, err = db.Id(inserted.ID.Hex()).UpdateOne(nil, bson.D{{Key: "name", Value: "bob"}})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	found, err = db.Id(inserted.ID.Hex()).FindOne(nil)
	if err != nil {
		t.Fatalf("find after update failed: %v", err)
	}
	if found == nil || found.Name != "bob" {
		t.Fatalf("unexpected value after update: %#v", found)
	}

	_, err = db.Id(inserted.ID.Hex()).DeleteOne(nil)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	found, err = db.Id(inserted.ID.Hex()).FindOne(nil)
	if err != nil {
		t.Fatalf("find after delete failed: %v", err)
	}
	if found != nil {
		t.Fatalf("entity should be deleted. got=%#v", found)
	}
}

func TestMongoIntegration_StringIDCRUD(t *testing.T) {
	client := mustIntegrationClient(t)
	db := NewDatabase[itStringModel](client)

	id := "str-" + time.Now().UTC().Format("20060102150405.000000000")
	entity := &itStringModel{BaseModelUUID: BaseModelUUID{ID: id}, Name: "alice"}
	inserted, err := db.InsertOne(entity)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if inserted.ID != id {
		t.Fatalf("string id mismatch. expected=%s got=%s", id, inserted.ID)
	}

	t.Cleanup(func() {
		_, _ = db.Id(id).DeleteOne(nil)
	})

	found, err := db.Id(id).FindOne(nil)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found == nil || found.Name != "alice" {
		t.Fatalf("unexpected find result: %#v", found)
	}

	_, err = db.Id(id).UpdateOne(nil, bson.D{{Key: "name", Value: "bob"}})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	found, err = db.Id(id).FindOne(nil)
	if err != nil {
		t.Fatalf("find after update failed: %v", err)
	}
	if found == nil || found.Name != "bob" {
		t.Fatalf("unexpected value after update: %#v", found)
	}
}

func mustIntegrationClient(t *testing.T) *Client {
	t.Helper()

	if testing.Short() {
		t.Skip("skip mongo integration test in short mode")
	}

	uri, dbName, err := resolveIntegrationMongoConfig()
	if err != nil {
		t.Skipf("mongo integration setup skipped: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	if err := mClient.Ping(ctx, nil); err != nil {
		t.Fatalf("ping mongo: %v", err)
	}
	_ = mClient.Disconnect(context.Background())

	client, err := NewClient(Config{URI: uri, Database: dbName})
	if err != nil {
		t.Fatalf("new goutil mongo client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func resolveIntegrationMongoConfig() (string, string, error) {
	uri := firstNonEmpty(
		strings.TrimSpace(os.Getenv("GOUTIL_IT_MONGO_URI")),
		strings.TrimSpace(os.Getenv("IT_MONGO_URI")),
	)
	dbName := firstNonEmpty(
		strings.TrimSpace(os.Getenv("GOUTIL_IT_MONGO_DB")),
		strings.TrimSpace(os.Getenv("IT_MONGO_DB")),
	)
	if uri != "" && dbName != "" {
		return uri, dbName, nil
	}

	if !useTestcontainersByDefault() {
		return "", "", fmt.Errorf("mongo config is missing and testcontainers is disabled")
	}

	tcOnce.Do(func() {
		tcErr = startMongoTestContainer()
	})
	if tcErr != nil {
		return "", "", tcErr
	}
	return tcMongoURI, tcMongoDB, nil
}

func startMongoTestContainer() error {
	timeoutSec := 60
	if raw := strings.TrimSpace(os.Getenv("GOUTIL_IT_TC_TIMEOUT_SEC")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeoutSec = parsed
		}
	}
	timeout := time.Duration(timeoutSec) * time.Second

	image := firstNonEmpty(
		strings.TrimSpace(os.Getenv("GOUTIL_IT_TC_MONGO_IMAGE")),
		"mongo:7",
	)
	dbName := firstNonEmpty(
		strings.TrimSpace(os.Getenv("GOUTIL_IT_TC_MONGO_DB")),
		"goutil_mongo_test",
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(timeout),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("start mongo testcontainer: %w", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(context.Background())
		return fmt.Errorf("resolve mongo testcontainer host: %w", err)
	}
	port, err := ctr.MappedPort(ctx, "27017/tcp")
	if err != nil {
		_ = ctr.Terminate(context.Background())
		return fmt.Errorf("resolve mongo testcontainer port: %w", err)
	}

	tcMongoCtr = ctr
	tcMongoURI = fmt.Sprintf("mongodb://%s:%s", host, port.Port())
	tcMongoDB = dbName
	return nil
}

func useTestcontainersByDefault() bool {
	value := strings.TrimSpace(os.Getenv("GOUTIL_IT_USE_TESTCONTAINERS"))
	if value == "" {
		// デフォルトは testcontainers モード
		return true
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
