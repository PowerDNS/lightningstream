package status

import (
	"context"
	"fmt"
	htmltemplate "html/template"
	"log"
	"net/http"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/wojas/go-healthz"

	"powerdns.com/platform/lightningstream/config"
)

func StartHTTPServer(c config.Config) {
	if c.HTTP.Address == "" {
		logrus.Info("HTTP stats server disabled")
		return
	}
	page := &Page{
		c: c,
	}
	logrus.WithField("address", c.HTTP.Address).Info("HTTP stats server enabled")
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthz", healthz.Handler())
	http.HandleFunc("/storage", page.BlobListPage)
	http.Handle("/", page)
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
		td.size       { text-align: right; }
		td.entries    { text-align: right; }
		td.error      { background-color: #ffb8b8; }
		td.no-error   { background-color: #a6f3a6; }
		a             { text-decoration: none; color: #3c6ac5; }
	</style>
</head>
<body>
	<h1>⚡ LightningStream Status</h1>
	<p>
		<a href="/metrics">Prometheus metrics</a>
		|
		<a href="/healthz">healthz</a>
	</p>

	<h2>LMDBs</h2>

	<table>
	<thead>
		<tr>
			<th>DB Name</th>
			<th>MapSize</th>
			<th>Used</th>
			<th>LastTxnID</th>
			<th>Readers</th>
		</tr>
	</thead>
	<tbody>
	{{range .DBInfo}}
		<tr>
			<td>{{.Name}}</td>
			<td class="size">{{byteSize .Info.MapSize}}</td>
			<td class="size">{{.Used.HumanReadable}}</td>
			<td class="size">{{.Info.LastTxnID}}</td>
			<td>{{.Info.NumReaders}} / {{.Info.MaxReaders}}</td>
		</tr>
	{{end}}
	</tbody>
	</table>

	{{range .DBInfo}}
		<h3>DBIs in {{.Name}}</h3>
		<table>
		<thead>
			<tr>
				<th>DBI</th>
				<th>Used</th>
				<th>Entries</th>
				<th>Depth</th>
				<th>BranchPages</th>
				<th>LeafPages</th>
				<th>OverflowPages</th>
			</tr>
		</thead>
		<tbody>
		{{range .DBIStats}}
			<tr>
				<td>{{.Name}}</td>
				<td class="size">{{.Used.HumanReadable}}</td>
				<td class="size">{{.Stat.Entries}}</td>
				<td class="size">{{.Stat.Depth}}</td>
				<td class="size">{{.Stat.BranchPages}}</td>
				<td class="size">{{.Stat.LeafPages}}</td>
				<td class="size">{{.Stat.OverflowPages}}</td>
			</tr>
		{{end}}
		</tbody>
		</table>
	{{end}}

	<h2>Storage</h2>
	<p><a href="storage">Storage snapshot listing (text)</a></p>

</body>
</html>`

var statusTemplate *htmltemplate.Template

func init() {
	var err error
	statusTemplate, err = htmltemplate.New("status").Funcs(htmltemplate.FuncMap{
		"byteSize": func(size int64) string {
			return datasize.ByteSize(size).HumanReadable()
		},
	}).Parse(statusTemplateString)
	if err != nil {
		log.Fatalf("BUG: Error in status HTML template: %v", err)
	}
}

func (p *Page) BlobListPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	list, err := gi.ListBlobs(ctx)
	if err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprintf(w, "ERROR LISTING STORAGE: %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	_, _ = fmt.Fprintf(w, "Name\tSize\n")
	for _, b := range list {
		_, _ = fmt.Fprintf(w, "%s\t%d\n", b.Name, b.Size)
	}
}

func (p *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Config config.Config
		DBInfo []DBInfo
	}{
		Config: p.c,
		DBInfo: gi.DBInfo(),
	}

	err := statusTemplate.Execute(w, data)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(fmt.Sprintf("Template execution error: %v", err)))
	}
}
