// Package jsonxmltool provides HTTP-oriented helpers for JSON, XML, uploads, and small web utilities.
package jsonxmltool

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// tokenAlphabet is the character set for generated opaque filenames and tokens.
const tokenAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321_+"

// defaultBodyLimit is the default maximum size (10 MiB) for JSON, XML, and upload payloads.
const defaultBodyLimit = 10 * 1024 * 1024

// Kit groups configurable limits and loggers for the helper methods below.
// Construct one with NewKit and call methods on a pointer when you need to mutate limits in place.
type Kit struct {
	MaxJSONSize        int         // cap on JSON request bodies; zero uses defaultBodyLimit
	MaxXMLSize         int         // cap on XML request bodies; zero uses defaultBodyLimit
	MaxFileSize        int         // cap per uploaded file in bytes; zero uses defaultBodyLimit
	AllowedFileTypes   []string    // MIME types allowed for uploads (e.g. image/png); empty allows any detected type
	AllowUnknownFields bool        // when false, json.Decoder rejects keys not present on the target struct
	ErrorLog           *log.Logger // diagnostics for failures
	InfoLog            *log.Logger // routine operational messages
}

// NewKit returns a Kit with 10 MiB limits and stdout loggers.
func NewKit() Kit {
	return Kit{
		MaxJSONSize: defaultBodyLimit,
		MaxXMLSize:  defaultBodyLimit,
		MaxFileSize: defaultBodyLimit,
		InfoLog:     log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		ErrorLog:    log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// JSONEnvelope is a conventional JSON API wrapper with optional data.
type JSONEnvelope struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    any `json:"data,omitempty"`
}

// XMLEnvelope mirrors JSONEnvelope for XML responses.
type XMLEnvelope struct {
	Error   bool        `xml:"error"`
	Message string      `xml:"message"`
	Data    any `xml:"data,omitempty"`
}

// ReadJSON decodes a single JSON value from r.Body into data (must be a pointer).
// When Content-Type is set, it must be application/json (case-insensitive).
func (k *Kit) ReadJSON(w http.ResponseWriter, r *http.Request, data any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.EqualFold(ct, "application/json") {
		return errors.New("Content-Type must be application/json")
	}

	maxBytes := defaultBodyLimit
	if k.MaxJSONSize != 0 {
		maxBytes = k.MaxJSONSize
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := k.newJSONDecoder(r.Body)

	if err := dec.Decode(data); err != nil {
		return k.mapJSONDecodeError(err, maxBytes)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}

// newJSONDecoder returns a decoder with Kit JSON policy applied.
func (k *Kit) newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	if !k.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}
	return dec
}

func (k *Kit) mapJSONDecodeError(err error, maxBytes int) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError

	switch {
	case errors.As(err, &syntaxError):
		return fmt.Errorf("malformed JSON near byte offset %d", syntaxError.Offset)
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("truncated JSON in request body")
	case errors.As(err, &unmarshalTypeError):
		return fmt.Errorf("JSON field %q has wrong type at offset %d", unmarshalTypeError.Field, unmarshalTypeError.Offset)
	case errors.Is(err, io.EOF):
		return errors.New("request body is empty")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("unexpected JSON field %s", fieldName)
	case err.Error() == "http: request body too large":
		return fmt.Errorf("JSON body exceeds limit of %d bytes", maxBytes)
	case errors.As(err, &invalidUnmarshalError):
		return fmt.Errorf("cannot unmarshal JSON into destination: %s", err.Error())
	default:
		return err
	}
}

// WriteJSON streams data as JSON via json.Encoder (no full-buffer Marshal).
// Optional headers merge into the response before the body is sent.
func (k *Kit) WriteJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	mergeHeaders(w, headers...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// ErrorJSON sends a JSONEnvelope with Error set and Message from err.
func (k *Kit) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	code := http.StatusBadRequest
	if len(status) > 0 {
		code = status[0]
	}
	payload := JSONEnvelope{Error: true, Message: err.Error()}
	return k.WriteJSON(w, code, payload)
}

// RandomString returns a string of length n drawn from tokenAlphabet.
func (k *Kit) RandomString(n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(tokenAlphabet)
	out := make([]rune, n)
	max := big.NewInt(int64(len(runes)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = runes[0]
			continue
		}
		out[i] = runes[idx.Int64()]
	}
	return string(out)
}

// PostJSON encodes data with json.Encoder into the POST body. An optional client overrides http.DefaultClient.
func (k *Kit) PostJSON(uri string, data any, client ...*http.Client) (*http.Response, int, error) {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(data); err != nil {
		return nil, 0, err
	}

	httpClient := &http.Client{}
	if len(client) > 0 && client[0] != nil {
		httpClient = client[0]
	}

	req, err := http.NewRequest(http.MethodPost, uri, &body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	return resp, resp.StatusCode, nil
}

// ServeAttachment streams a file from dir/name as a download using displayName in Content-Disposition.
func (k *Kit) ServeAttachment(w http.ResponseWriter, r *http.Request, dir, name, displayName string) {
	fp := path.Join(dir, name)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))
	http.ServeFile(w, r, fp)
}

// StoredFile records how an upload was persisted on disk.
type StoredFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// ReceiveOneUpload saves the first file in a multipart request to uploadDir.
func (k *Kit) ReceiveOneUpload(r *http.Request, uploadDir string, rename ...bool) (*StoredFile, error) {
	doRename := true
	if len(rename) > 0 {
		doRename = rename[0]
	}
	files, err := k.ReceiveUploads(r, uploadDir, doRename)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no files in upload")
	}
	return files[0], nil
}

// ReceiveUploads stores multipart files under uploadDir, optionally replacing names with random tokens.
func (k *Kit) ReceiveUploads(r *http.Request, uploadDir string, rename ...bool) ([]*StoredFile, error) {
	doRename := true
	if len(rename) > 0 {
		doRename = rename[0]
	}

	if err := k.EnsureDir(uploadDir); err != nil {
		return nil, err
	}

	maxSize := k.MaxFileSize
	if maxSize == 0 {
		maxSize = defaultBodyLimit
	}

	if err := r.ParseMultipartForm(int64(maxSize)); err != nil {
		return nil, fmt.Errorf("parse multipart form: %w", err)
	}

	var saved []*StoredFile
	for _, headers := range r.MultipartForm.File {
		for _, hdr := range headers {
			stored, err := func() (*StoredFile, error) {
				var rec StoredFile
				in, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer in.Close()

				if hdr.Size > int64(maxSize) {
					return nil, fmt.Errorf("file exceeds maximum size of %d bytes", maxSize)
				}

				sniff := make([]byte, 512)
				if _, err = in.Read(sniff); err != nil {
					return nil, err
				}

				detected := http.DetectContentType(sniff)
				if len(k.AllowedFileTypes) > 0 {
					allowed := false
					for _, want := range k.AllowedFileTypes {
						if strings.EqualFold(detected, want) {
							allowed = true
							break
						}
					}
					if !allowed {
						return nil, fmt.Errorf("content type %q is not permitted", detected)
					}
				}

				if _, err = in.Seek(0, io.SeekStart); err != nil {
					return nil, err
				}

				if doRename {
					rec.NewFileName = k.RandomString(25) + filepath.Ext(hdr.Filename)
				} else {
					rec.NewFileName = hdr.Filename
				}
				rec.OriginalFileName = hdr.Filename

				dest, err := os.Create(filepath.Join(uploadDir, rec.NewFileName))
				if err != nil {
					return nil, err
				}
				defer dest.Close()

				n, err := io.Copy(dest, in)
				if err != nil {
					return nil, err
				}
				rec.FileSize = n
				return &rec, nil
			}()
			if err != nil {
				return saved, err
			}
			saved = append(saved, stored)
		}
	}
	return saved, nil
}

// EnsureDir creates path and any missing parents with mode 0755.
func (k *Kit) EnsureDir(path string) error {
	const mode = 0o755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, mode)
	}
	return nil
}

// URLSlug lowercases s, replaces non-alphanumeric runs with hyphens, and trims edges.
func (k *Kit) URLSlug(s string) (string, error) {
	if s == "" {
		return "", errors.New("cannot slugify an empty string")
	}
	re := regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if slug == "" {
		return "", errors.New("slug would be empty after sanitizing")
	}
	return slug, nil
}

// WriteXML streams data with xml.Encoder after the standard XML declaration.
func (k *Kit) WriteXML(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	mergeHeaders(w, headers...)
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	return xml.NewEncoder(w).Encode(data)
}

// ReadXML decodes a single XML document from r.Body into data (must be a pointer).
func (k *Kit) ReadXML(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := defaultBodyLimit
	if k.MaxXMLSize != 0 {
		maxBytes = k.MaxXMLSize
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := xml.NewDecoder(r.Body)
	if err := dec.Decode(data); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain exactly one XML document")
	}
	return nil
}

// ErrorXML sends an XMLEnvelope with Error set and Message from err.
func (k *Kit) ErrorXML(w http.ResponseWriter, err error, status ...int) error {
	code := http.StatusBadRequest
	if len(status) > 0 {
		code = status[0]
	}
	payload := XMLEnvelope{Error: true, Message: err.Error()}
	return k.WriteXML(w, code, payload)
}

func mergeHeaders(w http.ResponseWriter, headers ...http.Header) {
	if len(headers) == 0 {
		return
	}
	for key, value := range headers[0] {
		w.Header()[key] = value
	}
}
