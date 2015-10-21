package main

import (
	"fmt"
	"runtime"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
)

func init() {
	runtime.LockOSThread()
}

var lastRenderCommands RenderCommandList
var engCmd EngineCommand
var event sdl.Event
var st *RenderCommandList
var rc *RenderCommand

func main() {
	fmt.Println("Starting up...")

	sdl.Init(sdl.INIT_EVERYTHING)

	// create window context
	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 1280, 720, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// create renderer context
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC) // |sdl.RENDERER_PRESENTVSYNC
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	// we're done loading the game, start the update loop
	sceneCh := SceneChannels{RCmd: make(chan *RenderCommandList, 1), Ev: make(chan Event, 256), Eng: make(chan EngineCommand), Err: make(chan error)}

	// load the gamescene and have it immediately start pumping out gamestates in a thread
	gameScene := GameScene{}
	go gameScene.Load(sceneCh)

	textures := make([]*sdl.Texture, 1024, 1024)
	curTex := 0

	//debug.SetGCPercent(-1)

	for {
		// process engine commands from the scene
		// scenes should block on waiting for the engine to return
		// if you want an engine function that doesn't block, just send a
		// response back on the channel immediately to ensure that all
		// calls from the scene can take the same procedure
		select {
		case engCmd = <-sceneCh.Eng:
			switch engCmd.Id {
			// load an image from disk and upload to gpu
			case EC_LOADIMAGE:
				fname := engCmd.Data.(string)
				image, err := img.Load(fname)
				if err != nil {
					fmt.Printf("Failed to load PNG: %s\n", err)
					return
				}
				defer image.Free()

				texture, err := renderer.CreateTextureFromSurface(image)
				if err != nil {
					panic("Error in CreateTextureFromSurface")
				}

				// FIXME: defer? need to delete these somewhere
				textures[curTex] = texture

				_, _, w, h, _ := texture.Query()
				sceneCh.Eng <- EngineCommand{Id: engCmd.Id, Success: true, Data: Image{Id: curTex, W: w, H: h}}
				curTex++
			}
		default:
		}

		// poll for input events and push them to the gamestate queue
		// this can technically fill the queue and block but it is very unlikely
		// FIXME: SDL_GetKeyboardState?
		for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				return
			case *sdl.MouseMotionEvent:
				sceneCh.Ev <- Event{Type: EV_MOUSEMOVE, Position: Vector{t.X, t.Y}}

			case *sdl.MouseButtonEvent:
				sceneCh.Ev <- Event{Type: EV_MOUSECLICK, Down: t.State != 0, EvData1: int(t.Button)}

			case *sdl.MouseWheelEvent:
				sceneCh.Ev <- Event{Type: EV_MOUSEWHEEL, Position: Vector{t.X, t.Y}}

			case *sdl.KeyDownEvent:
				sceneCh.Ev <- Event{Type: EV_KEY, Down: true, EvData1: int(t.Keysym.Scancode)}

			case *sdl.KeyUpEvent:
				sceneCh.Ev <- Event{Type: EV_KEY, Down: false, EvData1: int(t.Keysym.Scancode)}
			}
		}

		// check for new render lists

		// FIXME: may occasionally cause a black frame due to threading issues
		// we should probably request a render instead of polling
		// the lock doesn't seem to work or the copy isn't or something i don't know
		sceneCh.RCmdLock.Lock()
		select {
		case st = <-sceneCh.RCmd: // we have a new render command list
			lastRenderCommands = *st
		case err = <-sceneCh.Err: // we have an error from the gamestate
			return
		default:
		}

		// render whatever gamestate we have at the time
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()
		renderer.FillRect(&sdl.Rect{0, 0, 1280, 720})

		for i := 0; i < int(lastRenderCommands.NumCommands); i++ {
			rc = &lastRenderCommands.Commands[i]

			switch rc.Id {
			// load an image from disk and upload to gpu
			case RC_PIC:
				pic := rc.Data.(PicCommand)
				var srcRect *sdl.Rect
				if pic.SrcSize.W > 0 && pic.SrcSize.H > 0 {
					srcRect = &sdl.Rect{pic.SrcPos.X, pic.SrcPos.Y, pic.SrcSize.W, pic.SrcSize.H}
				}
				renderer.Copy(textures[pic.ImageId], srcRect, &sdl.Rect{pic.Pos.X, pic.Pos.Y, pic.Size.W, pic.Size.H})
			case RC_RECT:
				rect := rc.Data.(RectCommand)
				renderer.SetDrawColor(rect.Color.R, rect.Color.G, rect.Color.B, rect.Color.A)
				renderer.FillRect(&sdl.Rect{rect.Pos.X, rect.Pos.Y, rect.Size.W, rect.Size.H})
				renderer.SetDrawColor(0, 0, 0, 255)
			}
		}
		sceneCh.RCmdLock.Unlock()

		renderer.Present()
	}
}
