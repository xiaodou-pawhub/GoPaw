package builtin

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&ImageProcessTool{})
}

type ImageProcessTool struct {
	store   plugin.MediaStore
	session string
}

func (t *ImageProcessTool) Name() string { return "image_process" }

func (t *ImageProcessTool) Description() string {
	return "Process an image (resize, crop, rotate, grayscale). " +
		"Returns a new media reference. Actions: 'resize', 'crop', 'rotate', 'grayscale'."
}

func (t *ImageProcessTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "The media reference (media://uuid) or local path of the input image.",
			},
			"action": {
				Type:        "string",
				Description: "Action to perform.",
				Enum:        []string{"resize", "crop", "rotate", "grayscale"},
			},
			"width": {
				Type:        "integer",
				Description: "Width for resize/crop.",
			},
			"height": {
				Type:        "integer",
				Description: "Height for resize/crop.",
			},
			"angle": {
				Type:        "number",
				Description: "Rotation angle in degrees (clockwise).",
			},
		},
		Required: []string{"path", "action"},
	}
}

func (t *ImageProcessTool) SetMediaStore(s plugin.MediaStore) {
	t.store = s
}

func (t *ImageProcessTool) SetContext(channel, session, user string) {
	t.session = session
}

func (t *ImageProcessTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	action, _ := args["action"].(string)

	if t.store == nil {
		return plugin.ErrorResult("media store not initialized")
	}

	localPath := path
	if strings.HasPrefix(path, "media://") {
		var err error
		localPath, err = t.store.Resolve(path)
		if err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to resolve media: %v", err))
		}
	}

	// Load the image
	src, err := imaging.Open(localPath)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open image: %v", err))
	}

	var dst *image.NRGBA
	width, _ := args["width"].(float64)
	height, _ := args["height"].(float64)

	switch action {
	case "resize":
		if width <= 0 && height <= 0 {
			return plugin.ErrorResult("width or height must be > 0 for resize")
		}
		dst = imaging.Resize(src, int(width), int(height), imaging.Lanczos)
	case "crop":
		if width <= 0 || height <= 0 {
			return plugin.ErrorResult("width and height must be > 0 for crop")
		}
		dst = imaging.CenterCrop(src, int(width), int(height))
	case "rotate":
		angle, _ := args["angle"].(float64)
		dst = imaging.Rotate(src, angle, image.Transparent)
	case "grayscale":
		dst = imaging.Grayscale(src)
	default:
		return plugin.ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}

	// Save to a temporary file
	tmpPath := t.store.TempPath(".jpg")
	err = imaging.Save(dst, tmpPath, imaging.JPEGQuality(85))
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to save processed image: %v", err))
	}

	// Register in store
	newRef, err := t.store.Store(tmpPath, plugin.MediaMeta{
		Filename:    "processed_" + filepath.Base(tmpPath),
		ContentType: "image/jpeg",
		Source:      "tool:image_process",
	}, t.session) // Use session as scope for automatic cleanup

	if err != nil {
		_ = os.Remove(tmpPath)
		return plugin.ErrorResult(fmt.Sprintf("failed to register processed image: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("Image processed successfully. New reference: %s", newRef))
}
