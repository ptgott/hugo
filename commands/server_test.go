// Copyright 2015 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gohugoio/hugo/config"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/htesting"
	"github.com/gohugoio/hugo/hugofs"

	qt "github.com/frankban/quicktest"
)

func TestServer(t *testing.T) {
	if isWindowsCI() {
		// TODO(bep) not sure why server tests have started to fail on the Windows CI server.
		t.Skip("Skip server test on appveyor")
	}
	c := qt.New(t)
	dir, clean, err := createSimpleTestSite(t, testSiteConfig{})
	defer clean()
	c.Assert(err, qt.IsNil)

	// Let us hope that this port is available on all systems ...
	port := 1331

	defer func() {
		os.RemoveAll(dir)
	}()

	stop := make(chan bool)

	b := newCommandsBuilder()
	scmd := b.newServerCmdSignaled(stop)

	cmd := scmd.getCommand()
	cmd.SetArgs([]string{"-s=" + dir, fmt.Sprintf("-p=%d", port)})

	go func() {
		_, err = cmd.ExecuteC()
		c.Assert(err, qt.IsNil)
	}()

	// There is no way to know exactly when the server is ready for connections.
	// We could improve by something like https://golang.org/pkg/net/http/httptest/#Server
	// But for now, let us sleep and pray!
	time.Sleep(2 * time.Second)

	resp, err := http.Get("http://localhost:1331/")
	c.Assert(err, qt.IsNil)
	defer resp.Body.Close()
	homeContent := helpers.ReaderToString(resp.Body)

	c.Assert(homeContent, qt.Contains, "List: Hugo Commands")
	c.Assert(homeContent, qt.Contains, "Environment: development")

	// Stop the server.
	stop <- true
}

func TestFixURL(t *testing.T) {
	type data struct {
		TestName   string
		CLIBaseURL string
		CfgBaseURL string
		AppendPort bool
		Port       int
		Result     string
	}
	tests := []data{
		{"Basic http localhost", "", "http://foo.com", true, 1313, "http://localhost:1313/"},
		{"Basic https production, http localhost", "", "https://foo.com", true, 1313, "http://localhost:1313/"},
		{"Basic subdir", "", "http://foo.com/bar", true, 1313, "http://localhost:1313/bar/"},
		{"Basic production", "http://foo.com", "http://foo.com", false, 80, "http://foo.com/"},
		{"Production subdir", "http://foo.com/bar", "http://foo.com/bar", false, 80, "http://foo.com/bar/"},
		{"No http", "", "foo.com", true, 1313, "//localhost:1313/"},
		{"Override configured port", "", "foo.com:2020", true, 1313, "//localhost:1313/"},
		{"No http production", "foo.com", "foo.com", false, 80, "//foo.com/"},
		{"No http production with port", "foo.com", "foo.com", true, 2020, "//foo.com:2020/"},
		{"No config", "", "", true, 1313, "//localhost:1313/"},
	}

	for _, test := range tests {
		t.Run(test.TestName, func(t *testing.T) {
			b := newCommandsBuilder()
			s := b.newServerCmd()
			v := config.New()
			baseURL := test.CLIBaseURL
			v.Set("baseURL", test.CfgBaseURL)
			s.serverAppend = test.AppendPort
			s.serverPort = test.Port
			result, err := s.fixURL(v, baseURL, s.serverPort)
			if err != nil {
				t.Errorf("Unexpected error %s", err)
			}
			if result != test.Result {
				t.Errorf("Expected %q, got %q", test.Result, result)
			}
		})
	}
}

func TestRemoveErrorPrefixFromLog(t *testing.T) {
	c := qt.New(t)
	content := `ERROR 2018/10/07 13:11:12 Error while rendering "home": template: _default/baseof.html:4:3: executing "main" at <partial "logo" .>: error calling partial: template: partials/logo.html:5:84: executing "partials/logo.html" at <$resized.AHeight>: can't evaluate field AHeight in type *resource.Image
ERROR 2018/10/07 13:11:12 Rebuild failed: logged 1 error(s)
`

	withoutError := removeErrorPrefixFromLog(content)

	c.Assert(strings.Contains(withoutError, "ERROR"), qt.Equals, false)
}

func isWindowsCI() bool {
	return runtime.GOOS == "windows" && os.Getenv("CI") != ""
}

// Issue 7043
func TestServerReloadWithBadData(t *testing.T) {
	c := qt.New(t)
	dir, clean, err := htesting.CreateTempDir(hugofs.Os, "hugo-cli")

	cfgStr := `

baseURL = "https://example.org"
title = "Hugo Commands"

`
	os.MkdirAll(filepath.Join(dir, "public"), 0777)
	os.MkdirAll(filepath.Join(dir, "data"), 0777)
	os.MkdirAll(filepath.Join(dir, "layouts"), 0777)

	writeFile(t, filepath.Join(dir, "config.toml"), cfgStr)
	writeFile(t, filepath.Join(dir, "content", "p1.md"), `
---
title: "P1"
weight: 1
---

Content

`)
	defer clean()
	c.Assert(err, qt.IsNil)

	// The first time we write to the data file, the JSON is valid.
	writeFile(t, path.Join(dir, "data", "testdata.json"), `{
"key1": "value1",
"key2": "value2"
}`)

	writeFile(t, path.Join(dir, "layouts", "index.html"), `{{- range $k, $v := .Site.Data.testdata -}}
<p>{{$k}} has the value {{$v}}</p>
{{ end }}`)

	port := 1331

	b := newCommandsBuilder()
	stop := make(chan bool)
	scmd := b.newServerCmdSignaled(stop)

	defer func() {
		// Stop the server.
		stop <- true
	}()

	cmd := scmd.getCommand()
	cmd.SetArgs([]string{
		"-s=" + dir,
		fmt.Sprintf("-p=%d", port),
		"-d=" + path.Join(dir, "public"),
	})

	go func() {
		_, err = cmd.ExecuteC()
		c.Assert(err, qt.IsNil)
	}()

	// There is no way to know exactly when the server is ready for connections.
	// We could improve by something like https://golang.org/pkg/net/http/httptest/#Server
	// But for now, let us sleep and pray!
	time.Sleep(1 * time.Second)

	// Break the JSON by removing the comma
	writeFile(t, path.Join(dir, "data", "testdata.json"), `{
"key1": "value1"
"key2": "value2"
}`)

	// Wait for the server to make the change
	time.Sleep(1 * time.Second)

	// Fix the JSON and add a line
	writeFile(t, path.Join(dir, "data", "testdata.json"), `{
"key1": "value1",
"key2": "value2",
"key3": "value3"
}`)

	// Wait for the server to make the change
	time.Sleep(1 * time.Second)

	exp := `<p>key1 has the value value1</p>
<p>key2 has the value value2</p>
<p>key3 has the value value3</p>
`

	af, err := os.ReadFile(path.Join(dir, "public", "index.html"))
	c.Assert(err, qt.IsNil)
	c.Assert(string(af), qt.Equals, exp)

	return
}
