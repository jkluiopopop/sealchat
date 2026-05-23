package service

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func renderHTMLPart(payload *ExportPayload, assets viewerAssets) ([]byte, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload 不能为空")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化分片数据失败: %w", err)
	}
	return renderViewerShell("聊天分片", "window.__EXPORT_DATA__", data, assets, buildEmbeddedPlatformFontCSS(payload))
}

func renderViewerIndex(manifest *viewerManifest, assets viewerAssets) ([]byte, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest 不能为空")
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化 manifest 失败: %w", err)
	}
	return renderViewerShell("导出索引", "window.__EXPORT_INDEX__", data, assets, "")
}

func renderViewerShell(title, dataVar string, data []byte, assets viewerAssets, extraCSS string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html><html lang=\"zh\"><head><meta charset=\"UTF-8\">")
	buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">")
	buf.WriteString("<title>")
	buf.WriteString(htmlEscape(title))
	buf.WriteString("</title><style>")
	buf.WriteString(extraCSS)
	buf.WriteString(assets.CSS)
	buf.WriteString("</style></head><body><div id=\"app\"></div><script>")
	buf.WriteString(dataVar)
	buf.WriteString(" = ")
	buf.Write(data)
	buf.WriteString(";</script><script>")
	buf.WriteString(assets.JS)
	buf.WriteString("</script></body></html>")
	return buf.Bytes(), nil
}
