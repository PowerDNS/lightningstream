package status

import (
	"fmt"
	htmltemplate "html/template"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"powerdns.com/platform/lightningstream/config"
)

func StartHTTPServer(c config.Config) {
	if c.HTTP.Address == "" {
		logrus.Info("HTTP stats server disabled")
		return
	}
	logrus.WithField("address", c.HTTP.Address).Info("HTTP stats server enabled")
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", &Page{
		c: c,
	})
	go func() {
		err := http.ListenAndServe(c.HTTP.Address, nil)
		logrus.Fatalf("HTTP server error: %v", err)
	}()
}

type Page struct {
	c config.Config
}

const statusTemplateString = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>LightningStream Status</title>
	<style>
		body          { font-family: sans-serif; }
		table, td, th { border: 1px solid #ccc; border-collapse: collapse; }
		td, th        { padding: 5px; text-align: left; }
		td.last       { text-align: right; }
		td.error      { background-color: #ffb8b8; }
		td.no-error   { background-color: #a6f3a6; }
		a             { text-decoration: none; color: #3c6ac5; }
	</style>
</head>
<body>
	<h1>⚡ LightningStream Status</h1>
	<p>
		<a href="/metrics">Prometheus metrics</a>
	</p>

	<h2>Config</h2>
	<pre>{{ .Config.String }}</pre>

</body>
</html>`

var statusTemplate *htmltemplate.Template

func init() {
	var err error
	statusTemplate, err = htmltemplate.New("status").Parse(statusTemplateString)
	if err != nil {
		log.Fatalf("BUG: Error in status HTML template: %v", err)
	}
}

func (p *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Config config.Config
	}{
		Config: p.c,
	}

	err := statusTemplate.Execute(w, data)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(fmt.Sprintf("Template execution error: %v", err)))
	}
}
