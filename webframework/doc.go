package gw_web

import (
	"sort"
	"strings"
	"sync"
)

// RouteDoc describes a single endpoint for OpenAPI generation.
type RouteDoc struct {
	Summary     string
	Description string
	Tags        []string
	OperationID string
	Parameters  []ParamDoc
	RequestBody *RequestBodyDoc
	Responses   map[string]ResponseDoc // status code -> response
	Deprecated  bool
}

// ParamDoc describes a single OpenAPI parameter.
type ParamDoc struct {
	Name        string
	In          string // "query" | "path" | "header" | "cookie"
	Required    bool
	Description string
	Schema      SchemaDoc
	Example     any
}

// RequestBodyDoc describes a request body.
type RequestBodyDoc struct {
	Description string
	Required    bool
	Content     map[string]MediaTypeDoc // content-type -> schema
}

// MediaTypeDoc describes a single media type entry on request/response.
type MediaTypeDoc struct {
	Schema  SchemaDoc
	Example any
}

// ResponseDoc describes a single response.
type ResponseDoc struct {
	Description string
	Content     map[string]MediaTypeDoc
}

// SchemaDoc is a minimal OpenAPI schema descriptor sufficient for request/response shapes.
type SchemaDoc struct {
	Type        string // "string" | "integer" | "number" | "boolean" | "array" | "object"
	Format      string
	Description string
	Items       *SchemaDoc
	Properties  map[string]SchemaDoc
	Required    []string
	Ref         string // "#/components/schemas/<name>"; if set, other fields are ignored
	Enum        []any
}

// OpenAPIInfo describes the static metadata for an OpenAPI document.
type OpenAPIInfo struct {
	Title       string
	Version     string
	Description string
	Servers     []OpenAPIServer
}

// OpenAPIServer describes a server entry in an OpenAPI document.
type OpenAPIServer struct {
	URL         string
	Description string
}

// docRegistry collects route documentation registered on the WebApp.
type docRegistry struct {
	mu      sync.Mutex
	info    OpenAPIInfo
	entries []docEntry
	schemas map[string]SchemaDoc
}

type docEntry struct {
	method string
	path   string
	doc    RouteDoc
}

func newDocRegistry() *docRegistry {
	return &docRegistry{
		info: OpenAPIInfo{
			Title:   "API",
			Version: "0.0.0",
		},
		schemas: map[string]SchemaDoc{},
	}
}

func (r *docRegistry) register(method, path string, doc RouteDoc) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, docEntry{method: strings.ToLower(method), path: path, doc: doc})
}

// SetOpenAPIInfo replaces the static info block on the doc registry.
func (app WebApp) SetOpenAPIInfo(info OpenAPIInfo) {
	if app.docs == nil {
		return
	}
	app.docs.mu.Lock()
	defer app.docs.mu.Unlock()
	app.docs.info = info
}

// AddSchema registers a named component schema referenceable by SchemaDoc.Ref="#/components/schemas/<name>".
func (app WebApp) AddSchema(name string, schema SchemaDoc) {
	if app.docs == nil {
		return
	}
	app.docs.mu.Lock()
	defer app.docs.mu.Unlock()
	if app.docs.schemas == nil {
		app.docs.schemas = map[string]SchemaDoc{}
	}
	app.docs.schemas[name] = schema
}

// Doc methods on WebApp ////////////////////////////////////////////////////

// GetDoc registers a GET endpoint with OpenAPI documentation.
func (app WebApp) GetDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	app.Get(path, handlers...)
	app.docs.register(MethodGet, path, doc)
}

// PostDoc registers a POST endpoint with OpenAPI documentation.
func (app WebApp) PostDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	app.Post(path, handlers...)
	app.docs.register(MethodPost, path, doc)
}

// PutDoc registers a PUT endpoint with OpenAPI documentation.
func (app WebApp) PutDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	app.Put(path, handlers...)
	app.docs.register(MethodPut, path, doc)
}

// PatchDoc registers a PATCH endpoint with OpenAPI documentation.
func (app WebApp) PatchDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	app.Patch(path, handlers...)
	app.docs.register(MethodPatch, path, doc)
}

// DeleteDoc registers a DELETE endpoint with OpenAPI documentation.
func (app WebApp) DeleteDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	app.Delete(path, handlers...)
	app.docs.register(MethodDelete, path, doc)
}

// Doc methods on WebGroup //////////////////////////////////////////////////

// GetDoc registers a GET endpoint on the group with OpenAPI documentation.
func (group WebGroup) GetDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	group.Get(path, handlers...)
	group.docs.register(MethodGet, group.fullPath(path), doc)
}

// PostDoc registers a POST endpoint on the group with OpenAPI documentation.
func (group WebGroup) PostDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	group.Post(path, handlers...)
	group.docs.register(MethodPost, group.fullPath(path), doc)
}

// PutDoc registers a PUT endpoint on the group with OpenAPI documentation.
func (group WebGroup) PutDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	group.Put(path, handlers...)
	group.docs.register(MethodPut, group.fullPath(path), doc)
}

// PatchDoc registers a PATCH endpoint on the group with OpenAPI documentation.
func (group WebGroup) PatchDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	group.Patch(path, handlers...)
	group.docs.register(MethodPatch, group.fullPath(path), doc)
}

// DeleteDoc registers a DELETE endpoint on the group with OpenAPI documentation.
func (group WebGroup) DeleteDoc(path string, doc RouteDoc, handlers ...WebHandler) {
	group.Delete(path, handlers...)
	group.docs.register(MethodDelete, group.fullPath(path), doc)
}

func (group WebGroup) fullPath(path string) string {
	prefix := strings.TrimRight(group.prefix, "/")
	if path == "" {
		return prefix
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return prefix + path
}

// OpenAPI rendering ////////////////////////////////////////////////////////

// OpenAPI returns the collected route documentation as a map[string]any
// representing an OpenAPI 3.0.3 document. It is safe to call after registration completes.
func (app WebApp) OpenAPI() map[string]any {
	if app.docs == nil {
		return map[string]any{
			"openapi": "3.0.3",
			"info":    map[string]any{"title": "API", "version": "0.0.0"},
			"paths":   map[string]any{},
		}
	}
	return app.docs.toOpenAPI()
}

func (r *docRegistry) toOpenAPI() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := map[string]any{
		"title":   r.info.Title,
		"version": r.info.Version,
	}
	if r.info.Description != "" {
		info["description"] = r.info.Description
	}

	paths := map[string]any{}
	keys := make([]string, 0, len(r.entries))
	for _, e := range r.entries {
		keys = append(keys, e.method+" "+e.path)
	}
	sort.Strings(keys)

	for _, e := range r.entries {
		op := buildOperation(e.doc)
		if existing, ok := paths[e.path].(map[string]any); ok {
			existing[e.method] = op
			paths[e.path] = existing
		} else {
			paths[e.path] = map[string]any{e.method: op}
		}
	}

	doc := map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"paths":   paths,
	}
	if len(r.info.Servers) > 0 {
		servers := []map[string]any{}
		for _, s := range r.info.Servers {
			entry := map[string]any{"url": s.URL}
			if s.Description != "" {
				entry["description"] = s.Description
			}
			servers = append(servers, entry)
		}
		doc["servers"] = servers
	}
	if len(r.schemas) > 0 {
		schemas := map[string]any{}
		for name, s := range r.schemas {
			schemas[name] = schemaToMap(s)
		}
		doc["components"] = map[string]any{"schemas": schemas}
	}
	return doc
}

func buildOperation(doc RouteDoc) map[string]any {
	op := map[string]any{}
	if doc.Summary != "" {
		op["summary"] = doc.Summary
	}
	if doc.Description != "" {
		op["description"] = doc.Description
	}
	if doc.OperationID != "" {
		op["operationId"] = doc.OperationID
	}
	if len(doc.Tags) > 0 {
		op["tags"] = doc.Tags
	}
	if doc.Deprecated {
		op["deprecated"] = true
	}
	if len(doc.Parameters) > 0 {
		params := []map[string]any{}
		for _, p := range doc.Parameters {
			params = append(params, paramToMap(p))
		}
		op["parameters"] = params
	}
	if doc.RequestBody != nil {
		op["requestBody"] = requestBodyToMap(*doc.RequestBody)
	}
	responses := map[string]any{}
	if len(doc.Responses) == 0 {
		responses["200"] = map[string]any{"description": "OK"}
	} else {
		for code, r := range doc.Responses {
			responses[code] = responseToMap(r)
		}
	}
	op["responses"] = responses
	return op
}

func paramToMap(p ParamDoc) map[string]any {
	m := map[string]any{
		"name":     p.Name,
		"in":       p.In,
		"required": p.Required,
	}
	if p.Description != "" {
		m["description"] = p.Description
	}
	if !isEmptySchema(p.Schema) {
		m["schema"] = schemaToMap(p.Schema)
	}
	if p.Example != nil {
		m["example"] = p.Example
	}
	return m
}

func requestBodyToMap(b RequestBodyDoc) map[string]any {
	m := map[string]any{}
	if b.Description != "" {
		m["description"] = b.Description
	}
	if b.Required {
		m["required"] = true
	}
	if len(b.Content) > 0 {
		content := map[string]any{}
		for ct, mt := range b.Content {
			entry := map[string]any{}
			if !isEmptySchema(mt.Schema) {
				entry["schema"] = schemaToMap(mt.Schema)
			}
			if mt.Example != nil {
				entry["example"] = mt.Example
			}
			content[ct] = entry
		}
		m["content"] = content
	}
	return m
}

func responseToMap(r ResponseDoc) map[string]any {
	m := map[string]any{}
	if r.Description == "" {
		m["description"] = "OK"
	} else {
		m["description"] = r.Description
	}
	if len(r.Content) > 0 {
		content := map[string]any{}
		for ct, mt := range r.Content {
			entry := map[string]any{}
			if !isEmptySchema(mt.Schema) {
				entry["schema"] = schemaToMap(mt.Schema)
			}
			if mt.Example != nil {
				entry["example"] = mt.Example
			}
			content[ct] = entry
		}
		m["content"] = content
	}
	return m
}

func schemaToMap(s SchemaDoc) map[string]any {
	if s.Ref != "" {
		return map[string]any{"$ref": s.Ref}
	}
	m := map[string]any{}
	if s.Type != "" {
		m["type"] = s.Type
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.Items != nil {
		m["items"] = schemaToMap(*s.Items)
	}
	if len(s.Properties) > 0 {
		props := map[string]any{}
		for name, sub := range s.Properties {
			props[name] = schemaToMap(sub)
		}
		m["properties"] = props
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	return m
}

func isEmptySchema(s SchemaDoc) bool {
	return s.Type == "" && s.Ref == "" && s.Items == nil && len(s.Properties) == 0 && len(s.Enum) == 0 && s.Format == ""
}
