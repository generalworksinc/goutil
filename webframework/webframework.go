package gw_web

import (
	"context"
	"io"
	"log"
	"net"
	"time"

	"mime/multipart"

	"os"

	gw_errors "github.com/generalworksinc/goutil/errors"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
)

var store = session.NewStore() // v3: セッションのストア初期化は NewStore を使用

type WebCookie struct {
	Cookie interface{}
}
type WebCtx struct {
	Ctx interface{}
}
type WebApp struct {
	App interface{}
}
type WebGroup struct {
	Group interface{}
}
type WebRouter interface {
	Get(key string, defaultValue ...string) string
}
type WebHandler func(*WebCtx) error
type WsHandler func(*WebSocketConn)

type WebSocketConfig struct {
	Next              func(*WebCtx) bool
	HandshakeTimeout  time.Duration
	Subprotocols      []string
	Origins           []string
	ReadBufferSize    int
	WriteBufferSize   int
	EnableCompression bool
	RecoverHandler    func(*WebSocketConn)
}

func toFiberHandler(webHandler WebHandler) fiber.Handler {
	return func(fiberCtx fiber.Ctx) error {
		return webHandler(&WebCtx{Ctx: fiberCtx})
	}
}
func toFiberHandlers(webHandlerList []WebHandler) []any {
	hList := []any{}
	for _, handler := range webHandlerList {
		hList = append(hList, toFiberHandler(handler))
	}
	return hList
}

func toFiberHandlerFromWs(wsHandler WsHandler, cfg *WebSocketConfig) fiber.Handler {
	handler := func(conn *websocket.Conn) {
		wsHandler(&WebSocketConn{Conn: conn})
	}
	if cfg == nil {
		return websocket.New(handler)
	}
	fiberCfg := cfg.toFiberConfig()
	return websocket.New(handler, fiberCfg)
}

func toFiberHandlersFromWs(webHandlerList []WsHandler, cfg *WebSocketConfig) []any {
	hList := []any{}
	for _, handler := range webHandlerList {
		hList = append(hList, toFiberHandlerFromWs(handler, cfg))
	}
	return hList
}

// Application /////////////////////////////////////////////
func NewApp(errorHandler func(*WebCtx, error) error) *WebApp {
	fiberCfg := fiber.Config{
		//Prefork:       true,
		//CaseSensitive: true,
		//StrictRouting: true,
		//ServerHeader:  "Fiber",
		Immutable: true,               //安全側に倒す
		BodyLimit: 1024 * 1024 * 1024, //1 GB
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			return errorHandler(&WebCtx{Ctx: ctx}, err)
		},
	}

	app := fiber.New(fiberCfg)
	app.Use(compress.New())
	app.Use(cors.New())

	app.Use(func(c fiber.Ctx) (err error) {
		// Catch panics
		defer gw_errors.CatchPanic(&err, false) //このタイミングではエラーログをsentryに送信せず、Errorhandlerに任せる
		// return gw_errors.Wrap(err) if exist, else move to next handlerF
		return c.Next()
	})
	// v3: Static メソッドは削除。静的ミドルウェアに置き換え
	app.Use(static.New("/static", static.Config{FS: os.DirFS("static")}))
	return &WebApp{
		App: app,
	}
}

// formatをデフォルトを使う場合、nilをセット
func (app WebApp) SetLogger(writer io.Writer, format *string) {
	loggerConfig := logger.Config{Stream: writer}
	// NOTE: Fiber v3 の logger.Config は Output フィールドが廃止。出力の切り替えが必要な場合は
	// サービスや外部ロガー連携に移行する。ここでは Format のみ反映する。
	if format != nil {
		loggerConfig.Format = *format
	}

	app.App.(*fiber.App).Use(logger.New(loggerConfig))
}
func (app WebApp) Group(prefix string, handlers ...WebHandler) WebGroup {
	return WebGroup{
		Group: app.App.(*fiber.App).Group(prefix, toFiberHandlers(handlers)...),
	}
}
func (app WebApp) Get(path string, handlers ...WebHandler) {
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	app.App.(*fiber.App).Get(path, hs[0], hs[1:]...)
}
func (app WebApp) Post(path string, handlers ...WebHandler) {
	a := app.App.(*fiber.App)
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	a.Post(path, hs[0], hs[1:]...)
}
func (app WebApp) Put(path string, handlers ...WebHandler) {
	a := app.App.(*fiber.App)
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	a.Put(path, hs[0], hs[1:]...)
}
func (app WebApp) Patch(path string, handlers ...WebHandler) {
	a := app.App.(*fiber.App)
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	a.Patch(path, hs[0], hs[1:]...)
}
func (app WebApp) Delete(path string, handlers ...WebHandler) {
	a := app.App.(*fiber.App)
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	a.Delete(path, hs[0], hs[1:]...)
}
func (app WebApp) WsGet(path string, handlers ...WsHandler) {
	hs := toFiberHandlersFromWs(handlers, nil)
	if len(hs) == 0 {
		return
	}
	app.App.(*fiber.App).Get(path, hs[0], hs[1:]...)
}
func (app WebApp) WsGetWithConfig(path string, cfg WebSocketConfig, handlers ...WsHandler) {
	hs := toFiberHandlersFromWs(handlers, &cfg)
	if len(hs) == 0 {
		return
	}
	app.App.(*fiber.App).Get(path, hs[0], hs[1:]...)
}
func (app WebApp) Listen(addr string) error {
	a := app.App.(*fiber.App)
	return a.Listen(addr)
}
func (app WebApp) ShutdownWithTimeout(duration time.Duration) error {
	a := app.App.(*fiber.App)
	return a.ShutdownWithTimeout(duration)
}
func (app WebApp) ShutdownWithContext(ctx context.Context) error {
	a := app.App.(*fiber.App)
	return a.ShutdownWithContext(ctx)
}
func (app WebApp) Shutdown() error {
	a := app.App.(*fiber.App)
	return a.Shutdown()
}

// WebGroup ////////////////////////////////////////////////
func (group WebGroup) Get(path string, handlers ...WebHandler) {
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Get(path, hs[0], hs[1:]...)
}
func (group WebGroup) Post(path string, handlers ...WebHandler) {
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Post(path, hs[0], hs[1:]...)
}
func (group WebGroup) Put(path string, handlers ...WebHandler) {
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Put(path, hs[0], hs[1:]...)
}
func (group WebGroup) Patch(path string, handlers ...WebHandler) {
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Patch(path, hs[0], hs[1:]...)
}
func (group WebGroup) Delete(path string, handlers ...WebHandler) {
	hs := toFiberHandlers(handlers)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Delete(path, hs[0], hs[1:]...)
}
func (group WebGroup) Use(args ...interface{}) {
	convertedArgs := []interface{}{}
	for _, arg := range args {
		switch argType := arg.(type) {
		case WebHandler:
			log.Println("webHandler:", argType)
			convertedArgs = append(convertedArgs, toFiberHandler(arg.(WebHandler)))
		default:
			log.Println("not webHandler:", argType)
			convertedArgs = append(convertedArgs, arg)
		}
	}
	group.Group.(*fiber.Group).Use(convertedArgs...)
}

func (group WebGroup) WsGet(path string, handlers ...WsHandler) {
	hs := toFiberHandlersFromWs(handlers, nil)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Get(path, hs[0], hs[1:]...)
}
func (group WebGroup) WsGetWithConfig(path string, cfg WebSocketConfig, handlers ...WsHandler) {
	hs := toFiberHandlersFromWs(handlers, &cfg)
	if len(hs) == 0 {
		return
	}
	group.Group.(*fiber.Group).Get(path, hs[0], hs[1:]...)
}

// Cookie //////////////////////////////////////////////////
func (cookie WebCookie) SetName(val string) {
	cookie.Cookie.(*fiber.Cookie).Name = val
}
func (cookie WebCookie) SetValue(val string) {
	cookie.Cookie.(*fiber.Cookie).Value = val
}

// Session /////////////////////////////////////////////////

func (ctx WebCtx) SessionSet(key string, value string) error {
	sess, err := store.Get(ctx.Ctx.(fiber.Ctx))
	if err != nil {
		return gw_errors.Wrap(err)
	}
	defer sess.Release()
	sess.Set(key, value)
	return nil
}
func (ctx WebCtx) SessionGet(key string) interface{} {
	sess, err := store.Get(ctx.Ctx.(fiber.Ctx))
	if err != nil {
		return nil
	}
	defer sess.Release()
	return sess.Get(key)
}
func (ctx WebCtx) SessionSave() error {
	sess, err := store.Get(ctx.Ctx.(fiber.Ctx))
	if err != nil {
		return gw_errors.Wrap(err)
	}
	defer sess.Release()
	return gw_errors.Wrap(sess.Save())
}

// Context /////////////////////////////////////////////////
func (ctx WebCtx) Type(extension string, charset ...string) *WebCtx {
	ctx.Ctx = ctx.Ctx.(fiber.Ctx).Type(extension, charset...)
	return &ctx
}
func (ctx WebCtx) Send(body []byte) error {
	return ctx.Ctx.(fiber.Ctx).Send(body)
}
func (ctx WebCtx) SendString(bodyStr string) error {
	return ctx.Ctx.(fiber.Ctx).SendString(bodyStr)
}
func (ctx WebCtx) Set(key string, val string) {
	ctx.Ctx.(fiber.Ctx).Set(key, val)
}
func (ctx WebCtx) SetRequestQuery(key string, value string) {
	ctx.Ctx.(fiber.Ctx).RequestCtx().QueryArgs().Set(key, value)
}
func (ctx WebCtx) Redirect(location string, status ...int) error {
	r := ctx.Ctx.(fiber.Ctx).Redirect()
	if len(status) > 0 {
		r = r.Status(status[0])
	}
	return r.To(location)
}
func (ctx WebCtx) Cookie(cookie *WebCookie) {
	ctx.Ctx.(fiber.Ctx).Cookie(cookie.Cookie.(*fiber.Cookie))
}
func (ctx WebCtx) Query(key string, defaultValue ...string) string {
	return ctx.Ctx.(fiber.Ctx).Query(key, defaultValue...)
}
func (ctx WebCtx) Params(key string, defaultValue ...string) string {
	return ctx.Ctx.(fiber.Ctx).Params(key, defaultValue...)
}
func (ctx WebCtx) Locals(key interface{}, value ...interface{}) interface{} {
	return ctx.Ctx.(fiber.Ctx).Locals(key, value...)
}
func (ctx WebCtx) Next() error {
	return ctx.Ctx.(fiber.Ctx).Next()
}

func (ctx WebCtx) QueryParser(out interface{}) error {
	return ctx.Ctx.(fiber.Ctx).Bind().Query(out)
}

// Deprecated: Use BindJSON/BindForm/BindQuery.
func (ctx WebCtx) BodyParser(out interface{}) error {
	return ctx.Ctx.(fiber.Ctx).Bind().Body(out)
}

func (ctx WebCtx) BindJSON(out interface{}) error {
	return ctx.Ctx.(fiber.Ctx).Bind().JSON(out)
}
func (ctx WebCtx) BindForm(out interface{}) error {
	return ctx.Ctx.(fiber.Ctx).Bind().Form(out)
}
func (ctx WebCtx) BindQuery(out interface{}) error {
	return ctx.Ctx.(fiber.Ctx).Bind().Query(out)
}

func (ctx WebCtx) FormFile(key string) (*multipart.FileHeader, error) {
	return ctx.Ctx.(fiber.Ctx).FormFile(key)
}
func (ctx WebCtx) FormValue(key string, defaultValue ...string) string {
	return ctx.Ctx.(fiber.Ctx).FormValue(key, defaultValue...)
}

func (ctx WebCtx) Get(key string, defaultValue ...string) string {
	return ctx.Ctx.(fiber.Ctx).Get(key, defaultValue...)
}

func (ctx WebCtx) JSON(data interface{}) error {
	return ctx.Ctx.(fiber.Ctx).JSON(data)
}
func (ctx WebCtx) Cookies(key string, defaultValue ...string) string {
	return ctx.Ctx.(fiber.Ctx).Cookies(key, defaultValue...)
}
func (ctx WebCtx) StatusCode() int {
	return ctx.Ctx.(fiber.Ctx).Response().StatusCode()
}
func (ctx WebCtx) Status(status int) WebCtx {
	ctx.Ctx.(fiber.Ctx).Status(status)
	return ctx
}
func (ctx WebCtx) BaseURL() string {
	return ctx.Ctx.(fiber.Ctx).BaseURL()
}
func (ctx WebCtx) OriginalURL() string {
	return ctx.Ctx.(fiber.Ctx).OriginalURL()
}
func (ctx WebCtx) Method(override ...string) string {
	return ctx.Ctx.(fiber.Ctx).Method(override...)
}
func (ctx WebCtx) Protocol() string {
	return ctx.Ctx.(fiber.Ctx).Protocol()
}
func (ctx WebCtx) IP() string {
	return ctx.Ctx.(fiber.Ctx).IP()
}
func (ctx WebCtx) UserAgent() string {
	// return string(ctx.Ctx.(fiber.Ctx).RequestCtx().Request.Header.UserAgent())
	return ctx.Ctx.(fiber.Ctx).Get(fiber.HeaderUserAgent)
}
func (ctx WebCtx) SetHeader(key string, val string) {
	ctx.Ctx.(fiber.Ctx).Set(key, val)
}
func (ctx WebCtx) Body() []byte {
	return ctx.Ctx.(fiber.Ctx).Body()
}
func (ctx WebCtx) SendStream(stream io.Reader, size ...int) error {
	return ctx.Ctx.(fiber.Ctx).SendStream(stream, size...)
}
func (ctx WebCtx) BodyWriter() io.Writer {
	return ctx.Ctx.(fiber.Ctx).Response().BodyWriter()
}
func (ctx WebCtx) MultipartForm() (*multipart.Form, error) {
	return ctx.Ctx.(fiber.Ctx).MultipartForm()
}

func (ctx WebCtx) Attachment(filename ...string) {
	ctx.Ctx.(fiber.Ctx).Attachment(filename...)
}

func (ctx WebCtx) Done() <-chan struct{} {
	return ctx.Ctx.(fiber.Ctx).Done()
}

// WebSocket //////////////////////////////////////////////////
type WebSocketConn struct {
	Conn *websocket.Conn
}

func (conn *WebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	return conn.Conn.ReadMessage()
}
func (conn *WebSocketConn) WriteMessage(messageType int, data []byte) error {
	return conn.Conn.WriteMessage(messageType, data)
}
func (conn *WebSocketConn) Close() error {
	return conn.Conn.Close()
}
func (conn *WebSocketConn) NextReader() (messageType int, r io.Reader, err error) {
	return conn.Conn.NextReader()
}
func (conn *WebSocketConn) NextWriter(messageType int) (io.WriteCloser, error) {
	return conn.Conn.NextWriter(messageType)
}
func (conn *WebSocketConn) RemoteAddr() net.Addr {
	return conn.Conn.RemoteAddr()
}
func (conn *WebSocketConn) LocalAddr() net.Addr {
	return conn.Conn.LocalAddr()
}
func (conn *WebSocketConn) SetReadDeadline(t time.Time) error {
	return conn.Conn.SetReadDeadline(t)
}
func (conn *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return conn.Conn.SetWriteDeadline(t)
}
func (conn *WebSocketConn) SetPongHandler(handler func(appData string) error) {
	conn.Conn.SetPongHandler(handler)
}
func (conn *WebSocketConn) SetPingHandler(handler func(appData string) error) {
	conn.Conn.SetPingHandler(handler)
}
func (conn *WebSocketConn) SetCloseHandler(handler func(code int, text string) error) {
	conn.Conn.SetCloseHandler(handler)
}
func (conn *WebSocketConn) NetConn() net.Conn {
	return conn.Conn.NetConn()
}
func (conn *WebSocketConn) UnderlyingConn() net.Conn {
	return conn.Conn.UnderlyingConn()
}
func (conn *WebSocketConn) EnableWriteCompression(enable bool) {
	conn.Conn.EnableWriteCompression(enable)
}
func (conn *WebSocketConn) SetCompressionLevel(level int) error {
	return conn.Conn.SetCompressionLevel(level)
}
func (conn *WebSocketConn) CloseHandler() func(code int, text string) error {
	return conn.Conn.CloseHandler()
}
func (conn *WebSocketConn) PingHandler() func(appData string) error {
	return conn.Conn.PingHandler()
}
func (conn *WebSocketConn) PongHandler() func(appData string) error {
	return conn.Conn.PongHandler()
}
func (conn *WebSocketConn) Subprotocol() string {
	return conn.Conn.Subprotocol()
}
func (conn *WebSocketConn) Locals(key string, value ...interface{}) interface{} {
	return conn.Conn.Locals(key, value...)
}
func (conn *WebSocketConn) Params(key string, defaultValue ...string) string {
	return conn.Conn.Params(key, defaultValue...)
}
func (conn *WebSocketConn) Query(key string, defaultValue ...string) string {
	return conn.Conn.Query(key, defaultValue...)
}
func (conn *WebSocketConn) Cookies(key string, defaultValue ...string) string {
	return conn.Conn.Cookies(key, defaultValue...)
}
func (conn *WebSocketConn) Headers(key string, defaultValue ...string) string {
	return conn.Conn.Headers(key, defaultValue...)
}
func (conn *WebSocketConn) IP() string {
	return conn.Conn.IP()
}

func (cfg WebSocketConfig) toFiberConfig() websocket.Config {
	fiberCfg := websocket.Config{
		HandshakeTimeout:  cfg.HandshakeTimeout,
		Subprotocols:      cfg.Subprotocols,
		Origins:           cfg.Origins,
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		EnableCompression: cfg.EnableCompression,
	}
	if cfg.Next != nil {
		fiberCfg.Next = func(c fiber.Ctx) bool {
			return cfg.Next(&WebCtx{Ctx: c})
		}
	}
	if cfg.RecoverHandler != nil {
		fiberCfg.RecoverHandler = func(c *websocket.Conn) {
			cfg.RecoverHandler(&WebSocketConn{Conn: c})
		}
	}
	return fiberCfg
}

// HTTP Headers were copied from net/http.
const (
	HeaderAuthorization                   = "Authorization"
	HeaderProxyAuthenticate               = "Proxy-Authenticate"
	HeaderProxyAuthorization              = "Proxy-Authorization"
	HeaderWWWAuthenticate                 = "WWW-Authenticate"
	HeaderAge                             = "Age"
	HeaderCacheControl                    = "Cache-Control"
	HeaderClearSiteData                   = "Clear-Site-Data"
	HeaderExpires                         = "Expires"
	HeaderPragma                          = "Pragma"
	HeaderWarning                         = "Warning"
	HeaderAcceptCH                        = "Accept-CH"
	HeaderAcceptCHLifetime                = "Accept-CH-Lifetime"
	HeaderContentDPR                      = "Content-DPR"
	HeaderDPR                             = "DPR"
	HeaderEarlyData                       = "Early-Data"
	HeaderSaveData                        = "Save-Data"
	HeaderViewportWidth                   = "Viewport-Width"
	HeaderWidth                           = "Width"
	HeaderETag                            = "ETag"
	HeaderIfMatch                         = "If-Match"
	HeaderIfModifiedSince                 = "If-Modified-Since"
	HeaderIfNoneMatch                     = "If-None-Match"
	HeaderIfUnmodifiedSince               = "If-Unmodified-Since"
	HeaderLastModified                    = "Last-Modified"
	HeaderVary                            = "Vary"
	HeaderConnection                      = "Connection"
	HeaderKeepAlive                       = "Keep-Alive"
	HeaderAccept                          = "Accept"
	HeaderAcceptCharset                   = "Accept-Charset"
	HeaderAcceptEncoding                  = "Accept-Encoding"
	HeaderAcceptLanguage                  = "Accept-Language"
	HeaderCookie                          = "Cookie"
	HeaderExpect                          = "Expect"
	HeaderMaxForwards                     = "Max-Forwards"
	HeaderSetCookie                       = "Set-Cookie"
	HeaderAccessControlAllowCredentials   = "Access-Control-Allow-Credentials"
	HeaderAccessControlAllowHeaders       = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowMethods       = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowOrigin        = "Access-Control-Allow-Origin"
	HeaderAccessControlExposeHeaders      = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge             = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders     = "Access-Control-Request-Headers"
	HeaderAccessControlRequestMethod      = "Access-Control-Request-Method"
	HeaderOrigin                          = "Origin"
	HeaderTimingAllowOrigin               = "Timing-Allow-Origin"
	HeaderXPermittedCrossDomainPolicies   = "X-Permitted-Cross-Domain-Policies"
	HeaderDNT                             = "DNT"
	HeaderTk                              = "Tk"
	HeaderContentDisposition              = "Content-Disposition"
	HeaderContentEncoding                 = "Content-Encoding"
	HeaderContentLanguage                 = "Content-Language"
	HeaderContentLength                   = "Content-Length"
	HeaderContentLocation                 = "Content-Location"
	HeaderContentType                     = "Content-Type"
	HeaderForwarded                       = "Forwarded"
	HeaderVia                             = "Via"
	HeaderXForwardedFor                   = "X-Forwarded-For"
	HeaderXForwardedHost                  = "X-Forwarded-Host"
	HeaderXForwardedProto                 = "X-Forwarded-Proto"
	HeaderXForwardedProtocol              = "X-Forwarded-Protocol"
	HeaderXForwardedSsl                   = "X-Forwarded-Ssl"
	HeaderXUrlScheme                      = "X-Url-Scheme"
	HeaderLocation                        = "Location"
	HeaderFrom                            = "From"
	HeaderHost                            = "Host"
	HeaderReferer                         = "Referer"
	HeaderReferrerPolicy                  = "Referrer-Policy"
	HeaderUserAgent                       = "User-Agent"
	HeaderAllow                           = "Allow"
	HeaderServer                          = "Server"
	HeaderAcceptRanges                    = "Accept-Ranges"
	HeaderContentRange                    = "Content-Range"
	HeaderIfRange                         = "If-Range"
	HeaderRange                           = "Range"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderCrossOriginResourcePolicy       = "Cross-Origin-Resource-Policy"
	HeaderExpectCT                        = "Expect-CT"
	// Deprecated: use HeaderPermissionsPolicy instead
	HeaderFeaturePolicy           = "Feature-Policy"
	HeaderPermissionsPolicy       = "Permissions-Policy"
	HeaderPublicKeyPins           = "Public-Key-Pins"
	HeaderPublicKeyPinsReportOnly = "Public-Key-Pins-Report-Only"
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderUpgradeInsecureRequests = "Upgrade-Insecure-Requests"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXDownloadOptions        = "X-Download-Options"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderXPoweredBy              = "X-Powered-By"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderLastEventID             = "Last-Event-ID"
	HeaderNEL                     = "NEL"
	HeaderPingFrom                = "Ping-From"
	HeaderPingTo                  = "Ping-To"
	HeaderReportTo                = "Report-To"
	HeaderTE                      = "TE"
	HeaderTrailer                 = "Trailer"
	HeaderTransferEncoding        = "Transfer-Encoding"
	HeaderSecWebSocketAccept      = "Sec-WebSocket-Accept"
	HeaderSecWebSocketExtensions  = "Sec-WebSocket-Extensions"
	HeaderSecWebSocketKey         = "Sec-WebSocket-Key"
	HeaderSecWebSocketProtocol    = "Sec-WebSocket-Protocol"
	HeaderSecWebSocketVersion     = "Sec-WebSocket-Version"
	HeaderAcceptPatch             = "Accept-Patch"
	HeaderAcceptPushPolicy        = "Accept-Push-Policy"
	HeaderAcceptSignature         = "Accept-Signature"
	HeaderAltSvc                  = "Alt-Svc"
	HeaderDate                    = "Date"
	HeaderIndex                   = "Index"
	HeaderLargeAllocation         = "Large-Allocation"
	HeaderLink                    = "Link"
	HeaderPushPolicy              = "Push-Policy"
	HeaderRetryAfter              = "Retry-After"
	HeaderServerTiming            = "Server-Timing"
	HeaderSignature               = "Signature"
	HeaderSignedHeaders           = "Signed-Headers"
	HeaderSourceMap               = "SourceMap"
	HeaderUpgrade                 = "Upgrade"
	HeaderXDNSPrefetchControl     = "X-DNS-Prefetch-Control"
	HeaderXPingback               = "X-Pingback"
	HeaderXRequestID              = "X-Request-ID"
	HeaderXRequestedWith          = "X-Requested-With"
	HeaderXRobotsTag              = "X-Robots-Tag"
	HeaderXUACompatible           = "X-UA-Compatible"
)

// Network types that are commonly used
const (
	NetworkTCP  = "tcp"
	NetworkTCP4 = "tcp4"
	NetworkTCP6 = "tcp6"
)

// Compression types
const (
	StrGzip    = "gzip"
	StrBr      = "br"
	StrDeflate = "deflate"
	StrBrotli  = "brotli"
)

// Cookie SameSite
// https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-rfc6265bis-03#section-4.1.2.7
const (
	CookieSameSiteDisabled   = "disabled" // not in RFC, just control "SameSite" attribute will not be set.
	CookieSameSiteLaxMode    = "lax"
	CookieSameSiteStrictMode = "strict"
	CookieSameSiteNoneMode   = "none"
)

// HTTP methods were copied from net/http.
const (
	MethodGet     = "GET"     // RFC 7231, 4.3.1
	MethodHead    = "HEAD"    // RFC 7231, 4.3.2
	MethodPost    = "POST"    // RFC 7231, 4.3.3
	MethodPut     = "PUT"     // RFC 7231, 4.3.4
	MethodPatch   = "PATCH"   // RFC 5789
	MethodDelete  = "DELETE"  // RFC 7231, 4.3.5
	MethodConnect = "CONNECT" // RFC 7231, 4.3.6
	MethodOptions = "OPTIONS" // RFC 7231, 4.3.7
	MethodTrace   = "TRACE"   // RFC 7231, 4.3.8
	methodUse     = "USE"
)

// MIME types that are commonly used
const (
	MIMETextXML               = "text/xml"
	MIMETextHTML              = "text/html"
	MIMETextPlain             = "text/plain"
	MIMEApplicationXML        = "application/xml"
	MIMEApplicationJSON       = "application/json"
	MIMEApplicationJavaScript = "application/javascript"
	MIMEApplicationForm       = "application/x-www-form-urlencoded"
	MIMEOctetStream           = "application/octet-stream"
	MIMEMultipartForm         = "multipart/form-data"

	MIMETextXMLCharsetUTF8               = "text/xml; charset=utf-8"
	MIMETextHTMLCharsetUTF8              = "text/html; charset=utf-8"
	MIMETextPlainCharsetUTF8             = "text/plain; charset=utf-8"
	MIMEApplicationXMLCharsetUTF8        = "application/xml; charset=utf-8"
	MIMEApplicationJSONCharsetUTF8       = "application/json; charset=utf-8"
	MIMEApplicationJavaScriptCharsetUTF8 = "application/javascript; charset=utf-8"
)
