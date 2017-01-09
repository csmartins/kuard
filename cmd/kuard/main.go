/*
Copyright 2017 The KUAR Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/julienschmidt/httprouter"

	"github.com/jbeda/kuard/pkg/sitedata"
	"github.com/jbeda/kuard/pkg/version"
)

const serveAddr = ":8080"

func loggingMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

type pageContext struct {
	Version     string
	RequestDump string
	Env         map[string]string
}

type kuard struct {
	t *template.Template
}

func (k *kuard) getPageContext(r *http.Request) *pageContext {
	c := &pageContext{}
	c.Version = version.VERSION
	reqDump, _ := httputil.DumpRequest(r, false)
	c.RequestDump = string(reqDump)
	c.Env = map[string]string{}
	for _, e := range os.Environ() {
		splits := strings.SplitN(e, "=", 2)
		k, v := splits[0], splits[1]
		c.Env[k] = v
	}
	return c
}

func (k *kuard) rootHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	t := k.template("index.html")
	buf := &bytes.Buffer{}
	err := t.Execute(buf, k.getPageContext(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func (k *kuard) template(name string) *template.Template {
	if k.t == nil {
		k.t = k.loadTemplates()
	}
	t := k.t.Lookup(name)
	if t == nil {
		panic(fmt.Sprintf("Could not load template %v", name))
	}
	return t
}

func (k *kuard) loadTemplates() *template.Template {
	tFiles, err := sitedata.AssetDir("templates")
	if err != nil {
		panic(err)
	}

	t := template.New("")

	for _, tFile := range tFiles {
		fullName := path.Join("templates", tFile)
		data, err := sitedata.Asset(fullName)
		if err != nil {
			continue
		}
		log.Printf("Loading template for %v", tFile)
		_, err = t.New(tFile).Parse(string(data))
		if err != nil {
			log.Printf("ERROR: Could parse template %v: %v", tFile, err)
		}
	}
	return t
}

func main() {
	log.Printf("Starting kuard version: %v", version.VERSION)

	app := kuard{}

	router := httprouter.New()
	router.Handler("GET", "/static/*filepath", http.StripPrefix("/static/",
		http.FileServer(
			&assetfs.AssetFS{
				Asset:     sitedata.Asset,
				AssetDir:  func(path string) ([]string, error) { return nil, os.ErrNotExist },
				AssetInfo: sitedata.AssetInfo,
				Prefix:    "static",
			})))

	router.GET("/", app.rootHandler)

	log.Printf("Serving on %v", serveAddr)
	log.Fatal(http.ListenAndServe(serveAddr, loggingMiddleware(router)))
}
