package app

import (
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/macos"
)

// OpenWatermark opens the watermark configuration dialog for selected images in pane.
func (c *Coordinator) OpenWatermark(pane *PaneState) {
	if pane == nil {
		return
	}
	items := c.SelectedItems(pane)
	var imgPaths []string
	for _, it := range items {
		if !it.IsDir && macos.IsImageFile(it.Path) {
			imgPaths = append(imgPaths, it.Path)
		}
	}
	if len(imgPaths) == 0 {
		return
	}

	c.WatermarkPane = pane
	c.WatermarkError = ""
	c.WatermarkIsRunning = false
	c.WatermarkOptions = domain.DefaultWatermarkOptions()
	c.WatermarkOptions.SourceFiles = imgPaths
	c.WatermarkOptions.OutputDir = pane.Path
	c.ShowWatermarkModal = true
}

// CloseWatermark closes the watermark modal dialog.
func (c *Coordinator) CloseWatermark() {
	c.ShowWatermarkModal = false
	c.WatermarkError = ""
	c.WatermarkIsRunning = false
}

// ExecuteWatermark starts applying the configured watermark to all selected images.
func (c *Coordinator) ExecuteWatermark() {
	if c.WatermarkIsRunning || len(c.WatermarkOptions.SourceFiles) == 0 {
		return
	}

	c.WatermarkIsRunning = true
	c.WatermarkError = ""

	opts := c.WatermarkOptions
	targetPane := c.WatermarkPane

	go func() {
		err := macos.ApplyWatermark(opts)

		c.runOnMain(func() {
			c.WatermarkIsRunning = false
			if err != nil {
				c.WatermarkError = err.Error()
			} else {
				c.CloseWatermark()
				if targetPane != nil {
					c.LoadCurrentTab(nil)
				}
			}
		})
	}()
}
