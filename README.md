# go-json-xml-tool

A Go module of HTTP helpers focused on **JSON**, **XML**, multipart uploads, and small web utilities. Use it in handlers and services when you want consistent envelopes, body limits, and clearer decode errors without pulling in a full framework.

**Go:** 1.26+

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

Agent workflow files live under `docs/agent/` and `AGENTS.md`.
