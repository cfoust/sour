package preview

import (
	"image"
	"runtime"
	"sync"

	"github.com/rs/zerolog/log"
)

// RenderRequest is sent from worker goroutines to the main-thread render loop.
type RenderRequest struct {
	Preview *MapPreview
	Width   int
	Height  int
	Result  chan *image.RGBA
}

// RenderService proxies render requests to the main OS thread where the
// OpenGL context lives. Follows the same pattern as cy's opengl.Renderer.
type RenderService struct {
	commands chan RenderRequest
	quit     chan struct{}
	done     chan struct{}
}

var globalService *RenderService
var serviceOnce sync.Once

// NewRenderService creates the global render service. Call Poll() on the
// main goroutine to start processing. Safe to call multiple times.
func NewRenderService() {
	serviceOnce.Do(func() {
		globalService = &RenderService{
			commands: make(chan RenderRequest, 32),
			quit:     make(chan struct{}),
			done:     make(chan struct{}),
		}
	})
}

// Poll blocks on the current goroutine, locks it to the OS thread, creates
// the GL renderer, and processes render requests until Shutdown is called.
// Must be called from the main goroutine on macOS (GLFW requirement).
func Poll(width, height int) {
	if globalService == nil {
		return
	}
	svc := globalService

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	renderer := NewGLRenderer(width, height)
	if renderer != nil {
		log.Info().Msg("preview: GPU renderer active")
	} else {
		log.Info().Msg("preview: GPU unavailable, using software fallback")
	}

	for {
		select {
		case req := <-svc.commands:
			var img *image.RGBA
			if renderer != nil {
				img = renderer.RenderGL(req.Preview)
			}
			if img == nil {
				img = Render(req.Preview, RenderConfig{
					Width:  req.Width,
					Height: req.Height,
				})
			}
			req.Result <- img

		case <-svc.quit:
			if renderer != nil {
				renderer.Close()
			}
			close(svc.done)
			return
		}
	}
}

// Shutdown stops the render service. Blocks until Poll returns.
func Shutdown() {
	if globalService == nil {
		return
	}
	close(globalService.quit)
	<-globalService.done
}

// RenderPreview renders a preview, dispatching to the GPU service if running
// or falling back to software rendering. Safe to call from any goroutine.
func RenderPreview(p *MapPreview, width, height int) *image.RGBA {
	if globalService == nil {
		return Render(p, RenderConfig{Width: width, Height: height})
	}

	result := make(chan *image.RGBA, 1)
	globalService.commands <- RenderRequest{
		Preview: p,
		Width:   width,
		Height:  height,
		Result:  result,
	}
	return <-result
}
