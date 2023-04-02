package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/flosch/pongo2/v6"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
)

///////////////////////////////////////////////////////////////////////////////

func main() {
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index", pongo2.Context{
			"message": "Hello, world",
		})
	})
	//your actions here

	log.Fatal(e.Start(Listen))
}

///////////////////////////////////////////////////////////////////////////////

var (
	AssetsWebRoot      = "/assets"
	AssetsRoot         = "assets"
	TemplatesRoot      = "templates"
	TemplatesExtension = "twig"
	TemplatesDebug     = false
	Listen             = ":3000"
	e                  *echo.Echo
)

type pongo2Renderer struct{}

func init() {
	verbosePtr := flag.Bool("verbose", false, "show debug output")
	assetsWebRootPtr := flag.String("assets-web-root", "/assets", "url prefix for static assets")
	assetsRootPtr := flag.String("assets-root", "assets", "file system root for static assets")
	templatesRootPtr := flag.String("templates-root", "templates", "file system root for page templates")
	templatesExtensionPtr := flag.String("templates-extension", "twig", "page templates file extension")
	templatesDebugPtr := flag.Bool("templates-debug", false, "debug page templates (do not cache)")
	listenPtr := flag.String("listen", ":3000", "address and port to listen")
	//your flags here
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	if *verbosePtr {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	AssetsRoot = *assetsRootPtr
	AssetsWebRoot = *assetsWebRootPtr
	TemplatesRoot = *templatesRootPtr
	TemplatesExtension = *templatesExtensionPtr
	TemplatesDebug = *templatesDebugPtr
	Listen = *listenPtr

	e = echo.New()
	e.Renderer = pongo2Renderer{}
	e.HTTPErrorHandler = httpErrorHandler
	e.Static(AssetsWebRoot, AssetsRoot)
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			log.WithFields(log.Fields{
				"method": values.Method,
				"uri":    values.URI,
				"status": values.Status,
			}).Info("request")
			return nil
		},
	}))
}

func resolveTemplateName(n string) string {
	return fmt.Sprintf("%s%c%s.%s", TemplatesRoot, os.PathSeparator, n, TemplatesExtension)
}

func httpErrorHandler(e error, c echo.Context) {
	code := http.StatusInternalServerError
	if httpError, ok := e.(*echo.HTTPError); ok {
		code = httpError.Code
	}

	log.WithFields(log.Fields{
		"method": c.Request().Method,
		"uri":    c.Request().URL,
		"error":  e,
	}).Error("request error")

	if err := c.Render(code, "error", pongo2.Context{"error": e}); err != nil {
		log.Errorf("error page render error: %s", err)
	}
}

func (r pongo2Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	var ctx pongo2.Context
	var ok bool
	if data != nil {
		ctx, ok = data.(pongo2.Context)
		if !ok {
			return errors.New("no pongo2.Context data was passed")
		}
	}

	var t *pongo2.Template
	var err error
	if TemplatesDebug {
		t, err = pongo2.FromFile(resolveTemplateName(name))
	} else {
		t, err = pongo2.FromCache(resolveTemplateName(name))
	}
	if err != nil {
		return err
	}

	return t.ExecuteWriter(ctx, w)
}
