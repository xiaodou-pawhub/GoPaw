package builtin

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&ImageInfoTool{})
}

type ImageInfoTool struct {
	store plugin.MediaStore
}

func (t *ImageInfoTool) Name() string { return "image_info" }

func (t *ImageInfoTool) Description() string {
	return "Get metadata of an image, including width, height, format, and file size. " +
		"Use this before processing or sending large images."
}

func (t *ImageInfoTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "The media reference (media://uuid) or local path of the image.",
			},
		},
		Required: []string{"path"},
	}
}

func (t *ImageInfoTool) SetMediaStore(s plugin.MediaStore) {
	t.store = s
}

func (t *ImageInfoTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		return plugin.ErrorResult("path is required")
	}

	localPath := path
	if strings.HasPrefix(path, "media://") {
		if t.store == nil {
			return plugin.ErrorResult("media store not initialized")
		}
		var err error
		localPath, err = t.store.Resolve(path)
		if err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to resolve media: %v", err))
		}
	}

	file, err := os.Open(localPath)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open image: %v", err))
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to get file info: %v", err))
	}

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to decode image header: %v", err))
	}

	result := fmt.Sprintf("Image Metadata:\n- Format: %s\n- Dimensions: %dx%d\n- File Size: %d bytes",
		format, config.Width, config.Height, stat.Size())

	return plugin.NewToolResult(result)
}
