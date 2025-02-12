package gw_web

import (
	"io"
	"log"

	"mime/multipart"

	gw_errors "github.com/generalworksinc/goutil/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/session"
)

var store = session.New() // セッションのストアを初期化

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

func toFiberHandler(webHandler WebHandler) fiber.Handler {
	return func(fiberCtx *fiber.Ctx) error {
		return webHandler(&WebCtx{Ctx: fiberCtx})
	}
}
func toFiberHandlers(webHandlerList []WebHandler) []fiber.Handler {
	hList := []fiber.Handler{}
	for _, handler := range webHandlerList {
		hList = append(hList, toFiberHandler(handler))
	}
	return hList
}

// Application /////////////////////////////////////////////
func NewApp(errorHandler func(*WebCtx, error) error) *WebApp {
	app := fiber.New(fiber.Config{
		//Prefork:       true,
		//CaseSensitive: true,
		//StrictRouting: true,
		//ServerHeader:  "Fiber",
		BodyLimit: 1024 * 1024 * 1024, //1 GB
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return errorHandler(&WebCtx{Ctx: ctx}, err)
		},
	})
	app.Use(compress.New())
	app.Use(cors.New())

	app.Use(func(c *fiber.Ctx) (err error) {
		// Catch panics
		defer gw_errors.CatchPanic(&err, false) //このタイミングではエラーログをsentryに送信せず、Errorhandlerに任せる
		// Return err if exist, else move to next handlerF
		return c.Next()
	})
	app.Static("/static", "static")
	return &WebApp{
		App: app,
	}
}

// formatをデフォルトを使う場合、nilをセット
func (app WebApp) SetLogger(writer io.Writer, format *string) {
	loggerConfig := logger.Config{
		//TimeFormat: "02-Jan-2006",
		//TimeZone:   "America/New_York",
		Output: writer,
	}
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
	app.App.(*fiber.App).Get(path, toFiberHandlers(handlers)...)
}
func (app WebApp) Post(path string, handlers ...WebHandler) {
	a := app.App.(*fiber.App)
	a.Post(path, toFiberHandlers(handlers)...)
}
func (app WebApp) Listen(addr string) error {
	a := app.App.(*fiber.App)
	return a.Listen(addr)
}

// WebGroup ////////////////////////////////////////////////
func (group WebGroup) Get(path string, handlers ...WebHandler) {
	group.Group.(*fiber.Group).Get(path, toFiberHandlers(handlers)...)
}
func (group WebGroup) Post(path string, handlers ...WebHandler) {
	group.Group.(*fiber.Group).Post(path, toFiberHandlers(handlers)...)
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

// Cookie //////////////////////////////////////////////////
func (cookie WebCookie) SetName(val string) {
	cookie.Cookie.(*fiber.Cookie).Name = val
}
func (cookie WebCookie) SetValue(val string) {
	cookie.Cookie.(*fiber.Cookie).Value = val
}

// Session /////////////////////////////////////////////////

func (ctx WebCtx) SessionSet(key string, value string) error {
	sess, err := store.Get(ctx.Ctx.(*fiber.Ctx))
	if err != nil {
		return gw_errors.Wrap(err)
	}
	sess.Set(key, value)
	return nil
}
func (ctx WebCtx) SessionGet(key string) interface{} {
	sess, err := store.Get(ctx.Ctx.(*fiber.Ctx))
	if err != nil {
		return nil
	}
	return sess.Get(key)
}
func (ctx WebCtx) SessionSave() error {
	sess, err := store.Get(ctx.Ctx.(*fiber.Ctx))
	if err != nil {
		return gw_errors.Wrap(err)
	}
	return gw_errors.Wrap(sess.Save())
}

// Context /////////////////////////////////////////////////
func (ctx WebCtx) Type(extension string, charset ...string) *WebCtx {
	ctx.Ctx = ctx.Ctx.(*fiber.Ctx).Type(extension, charset...)
	return &ctx
}
func (ctx WebCtx) Send(body []byte) error {
	return ctx.Ctx.(*fiber.Ctx).Send(body)
}
func (ctx WebCtx) SendString(bodyStr string) error {
	return ctx.Ctx.(*fiber.Ctx).SendString(bodyStr)
}
func (ctx WebCtx) Set(key string, val string) {
	ctx.Ctx.(*fiber.Ctx).Set(key, val)
}
func (ctx WebCtx) Redirect(location string, status ...int) error {
	return ctx.Ctx.(*fiber.Ctx).Redirect(location, status...)
}
func (ctx WebCtx) Cookie(cookie *WebCookie) {
	ctx.Ctx.(*fiber.Ctx).Cookie(cookie.Cookie.(*fiber.Cookie))
}
func (ctx WebCtx) Query(key string, defaultValue ...string) string {
	return ctx.Ctx.(*fiber.Ctx).Query(key, defaultValue...)
}
func (ctx WebCtx) Params(key string, defaultValue ...string) string {
	return ctx.Ctx.(*fiber.Ctx).Params(key, defaultValue...)
}
func (ctx WebCtx) Locals(key interface{}, value ...interface{}) interface{} {
	return ctx.Ctx.(*fiber.Ctx).Locals(key, value...)
}
func (ctx WebCtx) Next() error {
	return ctx.Ctx.(*fiber.Ctx).Next()
}

func (ctx WebCtx) QueryParser(out interface{}) error {
	return ctx.Ctx.(*fiber.Ctx).QueryParser(out)
}
func (ctx WebCtx) BodyParser(out interface{}) error {
	return ctx.Ctx.(*fiber.Ctx).BodyParser(out)
}

func (ctx WebCtx) FormFile(key string) (*multipart.FileHeader, error) {
	return ctx.Ctx.(*fiber.Ctx).FormFile(key)
}
func (ctx WebCtx) FormValue(key string, defaultValue ...string) string {
	return ctx.Ctx.(*fiber.Ctx).FormValue(key, defaultValue...)
}

func (ctx WebCtx) Get(key string, defaultValue ...string) string {
	return ctx.Ctx.(*fiber.Ctx).Get(key, defaultValue...)
}

func (ctx WebCtx) JSON(data interface{}) error {
	return ctx.Ctx.(*fiber.Ctx).JSON(data)
}
func (ctx WebCtx) Cookies(key string, defaultValue ...string) string {
	return ctx.Ctx.(*fiber.Ctx).Cookies(key, defaultValue...)
}
func (ctx WebCtx) StatusCode() int {
	return ctx.Ctx.(*fiber.Ctx).Response().StatusCode()
}
func (ctx WebCtx) Status(status int) WebCtx {
	ctx.Ctx.(*fiber.Ctx).Status(status)
	return ctx
}
func (ctx WebCtx) BaseURL() string {
	return ctx.Ctx.(*fiber.Ctx).BaseURL()
}
func (ctx WebCtx) OriginalURL() string {
	return ctx.Ctx.(*fiber.Ctx).OriginalURL()
}
func (ctx WebCtx) Method(override ...string) string {
	return ctx.Ctx.(*fiber.Ctx).Method(override...)
}
func (ctx WebCtx) Protocol() string {
	return ctx.Ctx.(*fiber.Ctx).Protocol()
}
func (ctx WebCtx) IP() string {
	return ctx.Ctx.(*fiber.Ctx).IP()
}
func (ctx WebCtx) UserAgent() string {
	return string(ctx.Ctx.(*fiber.Ctx).Context().UserAgent())
}
func (ctx WebCtx) SetHeader(key string, val string) {
	ctx.Ctx.(*fiber.Ctx).Set(key, val)
}
func (ctx WebCtx) Body() []byte {
	return ctx.Ctx.(*fiber.Ctx).Body()
}
func (ctx WebCtx) SendStream(stream io.Reader, size ...int) error {
	return ctx.Ctx.(*fiber.Ctx).SendStream(stream, size...)
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
