package engine

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type FileUploadSecurity struct {
	mu                   sync.RWMutex
	devMode              bool
	maxFileSize          int64
	mimeSignatures       map[string][]byte
	executableExtensions []string
	archiveExtensions    []string
	imageExtensions      []string
	clamavAvailable      bool
	perEndpointLimits    map[string]int64
	executableMagic      [][]byte
	archiveMagic         [][]byte
}

func NewFileUploadSecurity(devMode bool) *FileUploadSecurity {
	u := &FileUploadSecurity{
		devMode:           devMode,
		maxFileSize:       10 * 1024 * 1024,
		clamavAvailable:   false,
		perEndpointLimits: make(map[string]int64),
	}

	u.mimeSignatures = map[string][]byte{
		"image/jpeg":               {0xFF, 0xD8, 0xFF},
		"image/png":                {0x89, 0x50, 0x4E, 0x47},
		"image/gif":                {0x47, 0x49, 0x46, 0x38},
		"image/webp":               {0x52, 0x49, 0x46, 0x46},
		"image/tiff":               {0x49, 0x49, 0x2A, 0x00},
		"image/bmp":                {0x42, 0x4D},
		"application/pdf":          {0x25, 0x50, 0x44, 0x46},
		"application/zip":          {0x50, 0x4B, 0x03, 0x04},
		"application/gzip":         {0x1F, 0x8B},
		"application/x-bzip2":      {0x42, 0x5A},
		"application/x-xz":         {0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
		"text/plain":               {},
		"text/csv":                 {},
		"application/json":         {},
		"application/xml":          {},
		"application/octet-stream": {},
	}

	u.executableExtensions = []string{
		".exe", ".dll", ".so", ".dylib", ".bin", ".bat", ".cmd",
		".com", ".msi", ".scr", ".pif", ".vbs", ".vbe", ".js",
		".jse", ".wsf", ".wsh", ".ps1", ".psm1", ".psd1",
		".msh", ".sh", ".bash", ".zsh", ".ksh", ".csh",
	}

	u.archiveExtensions = []string{
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".tgz", ".tbz2", ".zst", ".lz", ".lzma", ".lzo",
	}

	u.imageExtensions = []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp",
		".tiff", ".tif", ".svg", ".ico", ".heic", ".heif",
	}

	u.executableMagic = [][]byte{
		{0x7F, 0x45, 0x4C, 0x46},
		{0x4D, 0x5A},
		{0xCA, 0xFE, 0xBA, 0xBE},
		{0xCF, 0xFA, 0xED, 0xFE},
		{0xCE, 0xFA, 0xED, 0xFE},
		{0xFE, 0xED, 0xFA, 0xCE},
		{0xFE, 0xED, 0xFA, 0xCF},
		{0x23, 0x21},
	}

	u.archiveMagic = [][]byte{
		{0x50, 0x4B, 0x03, 0x04},
		{0x1F, 0x8B},
		{0x42, 0x5A},
		{0xFD, 0x37, 0x7A, 0x58, 0x5A},
		{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
		{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07},
		{0x75, 0x73, 0x74, 0x61, 0x72},
	}

	return u
}

func (u *FileUploadSecurity) Name() string { return "file_upload" }

func (u *FileUploadSecurity) Inspect(ctx *RequestContext) (*Decision, error) {
	contentType := ctx.Headers["Content-Type"]
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return nil, nil
	}

	contentLengthStr := ctx.Headers["Content-Length"]
	var contentLength int64
	if contentLengthStr != "" {
		if v, err := strconv.ParseInt(contentLengthStr, 10, 64); err == nil {
			contentLength = v
		}
	}

	limit := u.getEndpointLimit(ctx.Path)
	if contentLength > limit {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "UPL001",
			RuleName: "File Size Exceeded",
			Severity: "high",
			Score:    65,
			Evidence: fmt.Sprintf("file size %d exceeds limit %d for %s", contentLength, limit, ctx.Path),
		}, nil
	}

	if ctx.Body == nil || len(ctx.Body) == 0 {
		return nil, nil
	}

	boundary := u.extractBoundary(contentType)
	if boundary == "" {
		return nil, nil
	}

	reader := multipart.NewReader(bytes.NewReader(ctx.Body), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		part.Close()

		dec := u.inspectPart(part)
		if dec != nil {
			return dec, nil
		}
	}

	return nil, nil
}

func (u *FileUploadSecurity) inspectPart(part *multipart.Part) *Decision {
	filename := part.FileName()
	if filename == "" {
		return nil
	}

	contentType := part.Header.Get("Content-Type")

	var buf bytes.Buffer
	header := make([]byte, 512)
	n, _ := part.Read(header)
	if n > 0 {
		buf.Write(header[:n])
	}

	dec := u.verifyMIMEType(filename, contentType, buf.Bytes())
	if dec != nil {
		return dec
	}

	dec = u.detectExecutable(filename, buf.Bytes())
	if dec != nil {
		return dec
	}

	dec = u.detectArchiveBomb(filename, buf.Bytes())
	if dec != nil {
		return dec
	}

	dec = u.detectImagePolyglot(filename, buf.Bytes())
	if dec != nil {
		return dec
	}

	return nil
}

func (u *FileUploadSecurity) verifyMIMEType(filename, declaredType string, magic []byte) *Decision {
	if len(magic) == 0 {
		return nil
	}

	_ = strings.ToLower(filename[strings.LastIndex(filename, "."):])

	for _, sig := range u.mimeSignatures {
		if len(sig) > 0 && len(magic) >= len(sig) {
			if bytes.HasPrefix(magic, sig) {
				return nil
			}
		}
	}

	imageExt := false
	for _, ie := range u.imageExtensions {
		if strings.HasSuffix(strings.ToLower(filename), ie) {
			imageExt = true
			break
		}
	}

	if imageExt {
		isImage := false
		for _, sig := range u.mimeSignatures {
			if len(sig) > 0 && len(magic) >= len(sig) && bytes.HasPrefix(magic, sig) {
				isImage = true
				break
			}
		}
		if !isImage {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "UPL002",
				RuleName: "MIME Type Mismatch",
				Severity: "high",
				Score:    70,
				Evidence: fmt.Sprintf("file %s declared as %s but magic bytes mismatch", filename, declaredType),
			}
		}
	}

	return nil
}

func (u *FileUploadSecurity) detectExecutable(filename string, magic []byte) *Decision {
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	for _, execExt := range u.executableExtensions {
		if ext == execExt {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "UPL003",
				RuleName: "Executable Upload Blocked",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("executable file upload blocked: %s", filename),
			}
		}
	}

	for _, sig := range u.executableMagic {
		if len(magic) >= len(sig) && bytes.HasPrefix(magic, sig) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "UPL004",
				RuleName: "Executable Magic Bytes",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("executable magic bytes detected in %s", filename),
			}
		}
	}

	return nil
}

func (u *FileUploadSecurity) detectArchiveBomb(filename string, magic []byte) *Decision {
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	for _, archExt := range u.archiveExtensions {
		if ext == archExt {
			return &Decision{
				Action:   ActionMonitor,
				RuleID:   "UPL005",
				RuleName: "Archive Upload",
				Severity: "low",
				Score:    10,
				Evidence: fmt.Sprintf("archive upload: %s", filename),
			}
		}
	}

	for _, sig := range u.archiveMagic {
		if len(magic) >= len(sig) && bytes.HasPrefix(magic, sig) {
			ratio := 100 * len(magic) / len(sig)
			if ratio < 10 {
				return &Decision{
					Action:   ActionBlock,
					RuleID:   "UPL006",
					RuleName: "Archive Bomb Detected",
					Severity: "high",
					Score:    75,
					Evidence: fmt.Sprintf("potential archive bomb: compression ratio %d:1", ratio),
				}
			}
		}
	}

	return nil
}

func (u *FileUploadSecurity) detectImagePolyglot(filename string, magic []byte) *Decision {
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])

	isImageExt := false
	for _, ie := range u.imageExtensions {
		if ext == ie {
			isImageExt = true
			break
		}
	}

	if !isImageExt {
		return nil
	}

	if len(magic) < 4 {
		return nil
	}

	textPart := string(magic)

	scriptPattern := regexp.MustCompile(`(?i)(?:<script|<svg|onerror|onload|javascript:|<?php|<?xml)`)
	if scriptPattern.MatchString(textPart) {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "UPL007",
			RuleName: "Image Polyglot Detected",
			Severity: "critical",
			Score:    90,
			Evidence: fmt.Sprintf("image polyglot with script content in %s", filename),
		}
	}

	return nil
}

func (u *FileUploadSecurity) extractBoundary(contentType string) string {
	idx := strings.Index(strings.ToLower(contentType), "boundary=")
	if idx == -1 {
		return ""
	}
	boundary := contentType[idx+9:]
	if len(boundary) > 0 && boundary[0] == '"' {
		boundary = boundary[1:]
	}
	if idx := strings.IndexByte(boundary, '"'); idx != -1 {
		boundary = boundary[:idx]
	}
	return boundary
}

func (u *FileUploadSecurity) getEndpointLimit(path string) int64 {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if limit, ok := u.perEndpointLimits[path]; ok {
		return limit
	}
	return u.maxFileSize
}

func (u *FileUploadSecurity) SetEndpointLimit(path string, limit int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.perEndpointLimits[path] = limit
}

func (u *FileUploadSecurity) ScanWithClamAV(data []byte) *Decision {
	if !u.clamavAvailable {
		return nil
	}
	return nil
}


