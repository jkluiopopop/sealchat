package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	htmlnode "golang.org/x/net/html"

	"sealchat/model"
	"sealchat/service/storage"
)

var cssURLPattern = regexp.MustCompile(`url\(([^)]+)\)`)

type exportPlatformFontRef struct {
	ID     string
	Family string
}

func buildEmbeddedPlatformFontCSS(payload *ExportPayload) string {
	refs := collectPayloadPlatformFontRefs(payload)
	if len(refs) == 0 {
		return ""
	}
	blocks := make([]string, 0, len(refs))
	for _, ref := range refs {
		block, err := buildEmbeddedPlatformFontBlock(ref)
		if err != nil {
			continue
		}
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n")
}

func collectPayloadPlatformFontRefs(payload *ExportPayload) []exportPlatformFontRef {
	if payload == nil || len(payload.Messages) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	refs := make([]exportPlatformFontRef, 0)
	for _, msg := range payload.Messages {
		if strings.TrimSpace(msg.ContentHTML) == "" || !strings.Contains(msg.ContentHTML, "data-platform-font-id") {
			continue
		}
		nodes, err := htmlnode.ParseFragment(strings.NewReader(msg.ContentHTML), nil)
		if err != nil {
			continue
		}
		for _, node := range nodes {
			collectPlatformFontRefsFromNode(node, seen, &refs)
		}
	}
	return refs
}

func collectPlatformFontRefsFromNode(node *htmlnode.Node, seen map[string]struct{}, refs *[]exportPlatformFontRef) {
	if node == nil || refs == nil {
		return
	}
	if node.Type == htmlnode.ElementNode {
		fontID := ""
		family := ""
		for _, attr := range node.Attr {
			switch strings.ToLower(strings.TrimSpace(attr.Key)) {
			case "data-platform-font-id":
				fontID = strings.TrimSpace(attr.Val)
			case "data-platform-font-family":
				family = strings.TrimSpace(attr.Val)
			}
		}
		if fontID != "" {
			if _, ok := seen[fontID]; !ok {
				seen[fontID] = struct{}{}
				*refs = append(*refs, exportPlatformFontRef{ID: fontID, Family: family})
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectPlatformFontRefsFromNode(child, seen, refs)
	}
}

func buildEmbeddedPlatformFontBlock(ref exportPlatformFontRef) (string, error) {
	item, err := PlatformFontGet(ref.ID)
	if err != nil || item == nil || item.Status != model.PlatformFontStatusReady {
		return "", err
	}
	if item.DeliveryMode == model.PlatformFontDeliverySubset {
		if css, err := buildEmbeddedSubsetPlatformFontCSS(item); err == nil && strings.TrimSpace(css) != "" {
			return css, nil
		}
	}
	return buildEmbeddedSinglePlatformFontCSS(item, ref.Family)
}

func buildEmbeddedSubsetPlatformFontCSS(item *model.PlatformFontAsset) (string, error) {
	if item == nil || strings.TrimSpace(item.ManifestObjectKey) == "" {
		return "", fmt.Errorf("subset manifest missing")
	}
	manifestBytes, _, err := readPlatformFontObject(item.ManifestStorageType, item.ManifestObjectKey, "application/json")
	if err != nil {
		return "", err
	}
	var manifest PlatformFontSubsetManifestData
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return "", err
	}
	cssName := normalizePlatformFontSubsetPath(firstNonEmpty(manifest.CssName, manifest.Entry))
	if cssName == "" || !strings.HasSuffix(strings.ToLower(cssName), ".css") {
		return "", fmt.Errorf("subset css missing")
	}
	cssKey := storage.BuildPlatformFontSubsetObjectKey(item.ID, cssName)
	cssBytes, _, err := readPlatformFontObject(item.SubsetStorageType, cssKey, "text/css")
	if err != nil {
		return "", err
	}
	return rewritePlatformFontSubsetCSS(item, cssName, string(cssBytes), &manifest)
}

func rewritePlatformFontSubsetCSS(item *model.PlatformFontAsset, cssName string, cssText string, manifest *PlatformFontSubsetManifestData) (string, error) {
	if item == nil {
		return "", fmt.Errorf("nil item")
	}
	mimeByName := make(map[string]string)
	if manifest != nil {
		for _, chunk := range manifest.Chunks {
			name := normalizePlatformFontSubsetPath(chunk.Name)
			if name == "" {
				continue
			}
			mimeByName[name] = detectPlatformFontSubsetContentType(name, chunk.MimeType)
		}
	}
	var rewriteErr error
	rewritten := cssURLPattern.ReplaceAllStringFunc(cssText, func(match string) string {
		if rewriteErr != nil {
			return match
		}
		parts := cssURLPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		rawURL := strings.TrimSpace(parts[1])
		rawURL = strings.Trim(rawURL, `"'`)
		if rawURL == "" {
			return match
		}
		lower := strings.ToLower(rawURL)
		if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//") {
			return match
		}
		cleanURL := stripCSSURLDecorators(rawURL)
		resolvedName := resolveSubsetRelativePath(cssName, cleanURL)
		if resolvedName == "" {
			return match
		}
		objectKey := storage.BuildPlatformFontSubsetObjectKey(item.ID, resolvedName)
		data, mimeType, err := readPlatformFontObject(item.SubsetStorageType, objectKey, mimeByName[resolvedName])
		if err != nil {
			rewriteErr = err
			return match
		}
		if mimeType == "" {
			mimeType = detectPlatformFontSubsetContentType(resolvedName)
		}
		return fmt.Sprintf(`url("%s")`, buildDataURL(mimeType, data))
	})
	if rewriteErr != nil {
		return "", rewriteErr
	}
	return rewritten, nil
}

func buildEmbeddedSinglePlatformFontCSS(item *model.PlatformFontAsset, preferredFamily string) (string, error) {
	if item == nil || strings.TrimSpace(item.OriginalObjectKey) == "" {
		return "", fmt.Errorf("original font missing")
	}
	data, mimeType, err := readPlatformFontObject(item.OriginalStorageType, item.OriginalObjectKey, item.SourceMimeType)
	if err != nil {
		return "", err
	}
	family := strings.TrimSpace(preferredFamily)
	if family == "" {
		family = strings.TrimSpace(item.Family)
	}
	if family == "" {
		family = strings.TrimSpace(item.DisplayName)
	}
	if family == "" {
		return "", fmt.Errorf("font family missing")
	}
	src := fmt.Sprintf(`url("%s")`, buildDataURL(mimeType, data))
	if format := detectFontFormat(mimeType, item.SourceFileName); format != "" {
		src += fmt.Sprintf(` format("%s")`, format)
	}
	weight := normalizePlatformFontWeight(item.Weight)
	style := normalizePlatformFontStyle(item.Style)
	return fmt.Sprintf(`@font-face{font-family:%s;src:%s;font-weight:%s;font-style:%s;font-display:swap;}`, quoteCSSString(family), src, weight, style), nil
}

func readPlatformFontObject(storageType model.StorageType, objectKey string, fallbackContentType string) ([]byte, string, error) {
	if strings.TrimSpace(objectKey) == "" {
		return nil, "", fmt.Errorf("empty object key")
	}
	if storageType == model.StorageFontS3 {
		return fetchRemotePlatformFontObject(storageType, objectKey, fallbackContentType)
	}
	localPath, err := ResolveLocalPlatformFontPath(objectKey)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, "", err
	}
	contentType := strings.TrimSpace(fallbackContentType)
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectPlatformFontSubsetContentType(objectKey, fallbackContentType)
	}
	return data, contentType, nil
}

func fetchRemotePlatformFontObject(storageType model.StorageType, objectKey string, fallbackContentType string) ([]byte, string, error) {
	manager := GetStorageManager()
	if manager == nil {
		return nil, "", fmt.Errorf("storage manager not initialized")
	}
	target := manager.ResolveAttachmentExportURL(context.Background(), convertFontModelToBackend(storageType), objectKey)
	if strings.TrimSpace(target) == "" {
		return nil, "", fmt.Errorf("missing remote font url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("remote font fetch failed: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = fallbackContentType
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectPlatformFontSubsetContentType(objectKey, fallbackContentType)
	}
	return data, contentType, nil
}

func buildDataURL(contentType string, data []byte) string {
	mimeType := strings.TrimSpace(contentType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func stripCSSURLDecorators(value string) string {
	trimmed := strings.TrimSpace(value)
	if idx := strings.IndexAny(trimmed, "?#"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return trimmed
}

func resolveSubsetRelativePath(cssName string, target string) string {
	target = strings.TrimSpace(strings.ReplaceAll(target, "\\", "/"))
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "/") {
		return normalizePlatformFontSubsetPath(strings.TrimLeft(target, "/"))
	}
	baseDir := path.Dir(strings.ReplaceAll(cssName, "\\", "/"))
	if baseDir == "." {
		baseDir = ""
	}
	return normalizePlatformFontSubsetPath(path.Join(baseDir, target))
}

func detectFontFormat(mimeType string, fileName string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "font/woff2":
		return "woff2"
	case "font/woff":
		return "woff"
	case "font/ttf", "application/x-font-ttf":
		return "truetype"
	case "font/otf", "application/font-sfnt":
		return "opentype"
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".woff2":
		return "woff2"
	case ".woff":
		return "woff"
	case ".ttf":
		return "truetype"
	case ".otf":
		return "opentype"
	default:
		return ""
	}
}

func quoteCSSString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
