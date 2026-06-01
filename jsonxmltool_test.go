package jsonxmltool

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// roundTripFunc implements http.RoundTripper for tests without network I/O.
type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func testHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

type postBody struct {
	Data any `json:"bar"`
}

var postJSONCases = []struct {
	name    string
	payload any
	wantErr bool
}{
	{name: "encode ok", payload: postBody{Data: "bar"}, wantErr: false},
	{name: "encode fail", payload: make(chan int), wantErr: true},
}

func TestNewKit(t *testing.T) {
	k := NewKit()
	if k.MaxXMLSize != defaultBodyLimit {
		t.Errorf("MaxXMLSize = %d, want %d", k.MaxXMLSize, defaultBodyLimit)
	}
}

func TestKit_PostJSON(t *testing.T) {
	client := testHTTPClient(func(*http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("OK")),
			Header:     make(http.Header),
		}
	})

	for _, tc := range postJSONCases {
		t.Run(tc.name, func(t *testing.T) {
			var k Kit
			_, _, err := k.PostJSON("http://example.test/post", tc.payload, client)
			if (err != nil) != tc.wantErr {
				t.Fatalf("PostJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

var readJSONCases = []struct {
	name        string
	body        string
	wantErr     bool
	maxSize     int
	allowExtra  bool
	contentType string
}{
	{name: "valid object", body: `{"foo": "bar"}`, wantErr: false, maxSize: 1024},
	{name: "syntax error", body: `{"foo":"}`, wantErr: true, maxSize: 1024},
	{name: "type mismatch", body: `{"foo": 1}`, wantErr: true, maxSize: 1024},
	{name: "invalid json", body: `{1: 1}`, wantErr: true, maxSize: 1024},
	{name: "two values", body: `{"foo": "bar"}{"a": "b"}`, wantErr: true, maxSize: 1024},
	{name: "empty", body: ``, wantErr: true, maxSize: 1024},
	{name: "broken number", body: `{"foo": 1"}`, wantErr: true, maxSize: 1024},
	{name: "unknown field", body: `{"fooo": "bar"}`, wantErr: true, maxSize: 1024},
	{name: "wrong field type", body: `{"foo": 10.2}`, wantErr: true, maxSize: 1024},
	{name: "unknown allowed", body: `{"fooo": "bar"}`, wantErr: false, maxSize: 1024, allowExtra: true},
	{name: "unquoted key", body: `{jack: "bar"}`, wantErr: true, maxSize: 1024},
	{name: "over limit", body: `{"foo": "bar"}`, wantErr: true, maxSize: 5},
	{name: "not json", body: `Hello`, wantErr: true, maxSize: 1024},
	{name: "wrong content type", body: `{"foo": "bar"}`, wantErr: true, maxSize: 1024, contentType: "application/xml"},
}

func TestKit_ReadJSON(t *testing.T) {
	for _, tc := range readJSONCases {
		t.Run(tc.name, func(t *testing.T) {
			k := Kit{MaxJSONSize: tc.maxSize, AllowUnknownFields: tc.allowExtra}
			var got struct {
				Foo string `json:"foo"`
			}
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tc.body)))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			} else {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()
			err := k.ReadJSON(rr, req, &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestKit_ReadJSON_invalidDestination(t *testing.T) {
	var k Kit
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"foo":"bar"}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	if err := k.ReadJSON(rr, req, nil); err == nil {
		t.Fatal("expected error when destination is nil")
	}
}

var writeJSONCases = []struct {
	name    string
	payload any
	wantErr bool
}{
	{name: "ok", payload: JSONEnvelope{Message: "ok"}, wantErr: false},
	{name: "not encodable", payload: make(chan int), wantErr: true},
}

func TestKit_WriteJSON(t *testing.T) {
	for _, tc := range writeJSONCases {
		t.Run(tc.name, func(t *testing.T) {
			var k Kit
			rr := httptest.NewRecorder()
			h := http.Header{"X-Test": {"1"}}
			err := k.WriteJSON(rr, http.StatusOK, tc.payload, h)
			if (err != nil) != tc.wantErr {
				t.Fatalf("WriteJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestKit_ErrorJSON(t *testing.T) {
	var k Kit
	rr := httptest.NewRecorder()
	if err := k.ErrorJSON(rr, errors.New("boom"), http.StatusServiceUnavailable); err != nil {
		t.Fatal(err)
	}
	var env JSONEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if !env.Error || rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %+v status %d", env, rr.Code)
	}
}

func TestKit_RandomString(t *testing.T) {
	var k Kit
	if got := k.RandomString(12); len(got) != 12 {
		t.Fatalf("len = %d, want 12", len(got))
	}
}

func TestKit_ServeAttachment(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	var k Kit
	fixture := "./testdata/download-sample.jpg"
	info, err := os.Stat(fixture)
	if err != nil {
		t.Fatal(err)
	}

	k.ServeAttachment(rr, req, "./testdata", "download-sample.jpg", "jx-tool-fixture.jpg")
	res := rr.Result()
	defer res.Body.Close()

	wantLen := fmt.Sprintf("%d", info.Size())
	if res.Header.Get("Content-Length") != wantLen {
		t.Errorf("Content-Length = %q, want %q", res.Header.Get("Content-Length"), wantLen)
	}
	if res.Header.Get("Content-Disposition") != `attachment; filename="jx-tool-fixture.jpg"` {
		t.Errorf("Content-Disposition = %q", res.Header.Get("Content-Disposition"))
	}
	if _, err := io.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	}
}

var uploadCases = []struct {
	name       string
	allowed    []string
	rename     bool
	wantErr    bool
	maxSize    int
	uploadDir  string
}{
	{name: "keep name", allowed: []string{"image/jpeg", "image/png"}, rename: false, wantErr: false},
	{name: "random name", allowed: []string{"image/jpeg", "image/png"}, rename: true, wantErr: false},
	{name: "any mime", allowed: nil, rename: true, wantErr: false},
	{name: "reject mime", allowed: []string{"image/jpeg"}, wantErr: true},
	{name: "too large", allowed: []string{"image/png"}, wantErr: true, maxSize: 10},
	{name: "bad dir", allowed: []string{"image/png"}, wantErr: true, uploadDir: "//"},
}

func TestKit_ReceiveUploads(t *testing.T) {
	for _, tc := range uploadCases {
		t.Run(tc.name, func(t *testing.T) {
			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer writer.Close()
				defer wg.Done()
				part, err := writer.CreateFormFile("file", "upload-sample.png")
				if err != nil {
					t.Error(err)
					return
				}
				f, err := os.Open("./testdata/upload-sample.png")
				if err != nil {
					t.Error(err)
					return
				}
				defer f.Close()
				img, _, err := image.Decode(f)
				if err != nil {
					t.Error(err)
					return
				}
				if err := png.Encode(part, img); err != nil {
					t.Error(err)
				}
			}()

			req := httptest.NewRequest(http.MethodPost, "/", pr)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			k := Kit{AllowedFileTypes: tc.allowed}
			if tc.maxSize > 0 {
				k.MaxFileSize = tc.maxSize
			}
			dir := "./testdata/uploads/"
			if tc.uploadDir != "" {
				dir = tc.uploadDir
			}

			files, err := k.ReceiveUploads(req, dir, tc.rename)
			wg.Wait()

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			path := fmt.Sprintf("./testdata/uploads/%s", files[0].NewFileName)
			if _, err := os.Stat(path); err != nil {
				t.Fatal(err)
			}
			_ = os.Remove(path)
		})
	}
}

func TestKit_ReceiveOneUpload(t *testing.T) {
	cases := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{name: "ok", dir: "./testdata/uploads/", wantErr: false},
		{name: "bad dir", dir: "//", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)
			go func() {
				defer writer.Close()
				part, _ := writer.CreateFormFile("file", "upload-sample.png")
				f, _ := os.Open("./testdata/upload-sample.png")
				defer f.Close()
				img, _, _ := image.Decode(f)
				_ = png.Encode(part, img)
			}()
			req := httptest.NewRequest(http.MethodPost, "/", pr)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			k := Kit{AllowedFileTypes: []string{"image/png"}}
			file, err := k.ReceiveOneUpload(req, tc.dir, true)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			path := fmt.Sprintf("./testdata/uploads/%s", file.NewFileName)
			if _, err := os.Stat(path); err != nil {
				t.Fatal(err)
			}
			_ = os.Remove(path)
		})
	}
}

func TestKit_EnsureDir(t *testing.T) {
	var k Kit
	dir := "./testdata/tmpdir"
	if err := k.EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := k.EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(dir)
}

func TestKit_EnsureDir_forbidden(t *testing.T) {
	var k Kit
	if err := k.EnsureDir("/no-permission-root-dir"); err == nil {
		t.Fatal("expected error creating directory at filesystem root")
	}
}

var slugCases = []struct {
	name    string
	in      string
	want    string
	wantErr bool
}{
	{name: "words", in: "now is the time", want: "now-is-the-time"},
	{name: "empty", in: "", wantErr: true},
	{name: "punctuation", in: "Now is the time for all GOOD men! + Fish & such &^?123", want: "now-is-the-time-for-all-good-men-fish-such-123"},
	{name: "only cjk", in: "こんにちは世界", wantErr: true},
	{name: "mixed scripts", in: "こんにちは世界 hello world", want: "hello-world"},
}

func TestKit_URLSlug(t *testing.T) {
	var k Kit
	for _, tc := range slugCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := k.URLSlug(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("URLSlug() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKit_WriteXML(t *testing.T) {
	cases := []struct {
		name    string
		payload any
		wantErr bool
	}{
		{name: "ok", payload: XMLEnvelope{Message: "ok"}, wantErr: false},
		{name: "bad", payload: make(chan int), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var k Kit
			rr := httptest.NewRecorder()
			err := k.WriteXML(rr, http.StatusOK, tc.payload, http.Header{"X": {"y"}})
			if (err != nil) != tc.wantErr {
				t.Fatalf("WriteXML() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

var readXMLCases = []struct {
	name    string
	body    string
	max     int
	wantErr bool
}{
	{
		name: "valid",
		body: `<?xml version="1.0" encoding="UTF-8"?><note><to>A</to><from>B</from></note>`,
	},
	{name: "broken", body: `<?xml version="1.0"?><note><to>A</to><from>B</note>`, wantErr: true},
	{name: "too big", body: `<?xml version="1.0"?><note><to>A</to><from>B</from></note>`, max: 10, wantErr: true},
	{
		name:    "two docs",
		body:    `<?xml version="1.0"?><note><to>A</to></note><?xml version="1.0"?><note><to>B</to></note>`,
		wantErr: true,
	},
}

func TestKit_ReadXML(t *testing.T) {
	for _, tc := range readXMLCases {
		t.Run(tc.name, func(t *testing.T) {
			k := Kit{}
			if tc.max != 0 {
				k.MaxXMLSize = tc.max
			}
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tc.body)))
			rr := httptest.NewRecorder()
			var note struct {
				To   string `xml:"to"`
				From string `xml:"from"`
			}
			err := k.ReadXML(rr, req, &note)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadXML() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestKit_ErrorXML(t *testing.T) {
	var k Kit
	rr := httptest.NewRecorder()
	if err := k.ErrorXML(rr, errors.New("fail"), http.StatusServiceUnavailable); err != nil {
		t.Fatal(err)
	}
	var env XMLEnvelope
	if err := xml.NewDecoder(rr.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if !env.Error || rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %+v code %d", env, rr.Code)
	}
}
