[![Version](https://img.shields.io/badge/goversion-1.26.x-blue.svg)](https://go.dev/dl/)
<a href="https://go.dev"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with Go"></a>
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/dunky-star/go-json-xml-tool/blob/main/LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/dunky-star/go-json-xml-tool)](https://goreportcard.com/report/github.com/dunky-star/go-json-xml-tool)
[![Tests](https://github.com/dunky-star/go-json-xml-tool/actions/workflows/tests.yml/badge.svg)](https://github.com/dunky-star/go-json-xml-tool/actions/workflows/tests.yml)
[![pkg.go.dev reference](https://pkg.go.dev/badge/github.com/dunky-star/go-json-xml-tool.svg)](https://pkg.go.dev/github.com/dunky-star/go-json-xml-tool)
[![Coverage](https://img.shields.io/badge/coverage-92%25-brightgreen)](https://github.com/dunky-star/go-json-xml-tool/actions/workflows/tests.yml)

# GO HTTP JSON/XML -AND- FILE LIBRARY

A Go module of HTTP helpers focused on ***JSON***, ***XML***, multipart uploads, and small web utilities. Use it in handlers and services when you want consistent envelopes, body limits, and clearer decode errors without pulling in a full framework.

This is a **utility package**, not a web framework. It does not provide routing, middleware stacks, auth, validation libraries, or OpenAPI tooling.

## When to use

- You build APIs with ***`net/http`*** (or a thin router) and want shared handler patterns.
- You want ***body size limits***, stricter JSON parsing (unknown fields, single document), and ***human-readable decode errors*** without copying that logic into every handler.
- You want ***consistent JSON/XML response shapes*** (`JSONEnvelope`, `XMLEnvelope`) across endpoints.
- You need small, related helpers in the same style: ***multipart uploads***, ***attachment downloads***, ***URL slugs***, ***outbound JSON POSTs***.

The JSON handler example below is most valuable when you have ***many endpoints*** repeating the same read <> validate <> respond flow. For a single trivial handler, the standard library alone is often enough.

## When not to use

- You already use ***Gin, Echo, Fiber, Chi with binding***, or similar and get request parsing, limits, and errors from that stack.
- You need a ***full framework*** (routing, DI, migrations, generated API docs).
- Your service is ***not HTTP-facing***, or only needs one-off `json.Marshal` / `json.Unmarshal` with no shared conventions.

## Install

```bash
go get github.com/dunky-star/go-json-xml-tool
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/dunky-star/go-json-xml-tool"
)

func main() {
	k := jsonxmltool.NewKit()
	fmt.Println(k.RandomString(16))
}
```

## Encoding model

Reads use **`json.Decoder` / `xml.Decoder`** on the request body (streaming, size-limited).  
Writes use **`json.Encoder` / `xml.Encoder`** on the response or buffer — not `Marshal` — so large payloads are not held entirely in memory before send.

## What’s included

| Area | Methods |
|------|---------|
| JSON | `ReadJSON`, `WriteJSON`, `ErrorJSON`, `PostJSON` |
| XML | `ReadXML`, `WriteXML`, `ErrorXML` |
| Uploads | `ReceiveUploads`, `ReceiveOneUpload` |
| Files | `ServeAttachment`, `EnsureDir` |
| Text | `URLSlug`, `RandomString` |

Configure limits and MIME allowlists on `Kit`:

```go
k := jsonxmltool.NewKit()
k.MaxJSONSize = 1 << 20
k.AllowedFileTypes = []string{"image/png", "image/jpeg"}
```

## JSON handler example

```go
type createInput struct {
	Name string `json:"name"`
}

func create(w http.ResponseWriter, r *http.Request) {
	k := jsonxmltool.NewKit()
	var in createInput
	if err := k.ReadJSON(w, r, &in); err != nil {
		_ = k.ErrorJSON(w, err)
		return
	}
	_ = k.WriteJSON(w, http.StatusCreated, jsonxmltool.JSONEnvelope{
		Error: false, Message: "created", Data: in.Name,
	})
}
```

## Multipart upload example

Client form (any field name; `multiple` is optional):

```html
<form action="/upload" method="post" enctype="multipart/form-data">
  <input type="file" name="file" multiple />
  <button type="submit">Upload</button>
</form>
```

Handler:

```go
func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	k := jsonxmltool.Kit{
		MaxFileSize:      10 << 20, // 10 MiB
		AllowedFileTypes: []string{"image/png", "image/jpeg"},
	}

	// ReceiveUploads creates uploadDir if needed and assigns random on-disk names by default.
	// Pass false as the third argument to keep original filenames: ReceiveUploads(r, "./uploads", false)
	files, err := k.ReceiveUploads(r, "./uploads")
	if err != nil {
		_ = k.ErrorJSON(w, err)
		return
	}

	_ = k.WriteJSON(w, http.StatusOK, jsonxmltool.JSONEnvelope{
		Error:   false,
		Message: "uploaded",
		Data:    files, // []*jsonxmltool.StoredFile (saved name, original name, size)
	})
}

// Single file only:
func uploadOne(w http.ResponseWriter, r *http.Request) {
	k := jsonxmltool.NewKit()
	file, err := k.ReceiveOneUpload(r, "./uploads")
	if err != nil {
		_ = k.ErrorJSON(w, err)
		return
	}
	_ = k.WriteJSON(w, http.StatusOK, jsonxmltool.JSONEnvelope{Data: file})
}
```

`ReceiveUploads` checks each file’s size against `MaxFileSize`, sniffs MIME type (first 512 bytes), and rejects types not listed in `AllowedFileTypes` (empty list allows any detected type).

## XML handler example

```go
import (
	"encoding/xml"
	"net/http"

	"github.com/dunky-star/go-json-xml-tool"
)

type ping struct {
	XMLName xml.Name `xml:"ping"`
	Text    string   `xml:",chardata"`
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	k := jsonxmltool.NewKit()
	var body ping
	if err := k.ReadXML(w, r, &body); err != nil {
		_ = k.ErrorXML(w, err)
		return
	}
	_ = k.WriteXML(w, http.StatusOK, jsonxmltool.XMLEnvelope{Message: "pong"})
}
```

## Development

```bash
go test ./...
go vet ./...
```
