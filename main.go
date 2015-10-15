package main

import (
	"fmt"
	"runtime"

	"github.com/veandco/go-sdl2/sdl"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	fmt.Println("Starting up...")

	sdl.Init(sdl.INIT_EVERYTHING)

	// create window context
	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN)
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
	sceneCh := SceneChannels{Gs: make(chan *GameState, 1), Ev: make(chan Event, 256), Eng: make(chan EngineCommand), Err: make(chan error)}

	// load the gamescene and have it immediately start pumping out gamestates in a thread
	gameScene := GameScene{}
	go gameScene.Load(sceneCh)

	var st *GameState
	textures := make([]*sdl.Texture, 1024, 1024)
	curTex := 0

	for {
		// process engine commands from the scene
		// scenes should block on waiting for the engine to return
		// if you want an engine function that doesn't block, just send a
		// response back on the channel immediately to ensure that all
		// calls from the scene can take the same procedure
		select {
		case engCmd := <-sceneCh.Eng:
			switch engCmd.Id {
			// upload an sdl.Image to the GPU
			// FIXME: the gamestate shouldn't use sdl, lets just pass a path to the engine
			// and have the engine just do the file i/o on the main thread
			case EC_UPLOADIMAGE:
				image := engCmd.Data.(*sdl.Surface)
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
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
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

		// check for new gamestates
		select {
		case st = <-sceneCh.Gs: // we have a new gamestate
		case err = <-sceneCh.Err: // we have an error from the gamestate
			return
		default:
		}

		// FIXME: we should probably draw a loading here instead, or have the gamescene immediately
		// send down a loading state
		if st == nil {
			continue
		}

		// render whatever gamestate we have at the time
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()
		// for now, all entities are just rectangles, so draw based on ent.pos and ent.size
		// FIXME: the engine should read a list of rendering commands, not the gamestate
		for _, ent := range st.Entities {
			if !ent.Valid {
				continue
			}

			renderer.SetDrawColor(ent.Color.R, ent.Color.G, ent.Color.B, ent.Color.A)
			if textures[ent.Image] != nil {
				renderer.Copy(textures[ent.Image], nil, &sdl.Rect{ent.Pos.X, ent.Pos.Y, ent.Size.W, ent.Size.H})
			} else {
				renderer.FillRect(&sdl.Rect{ent.Pos.X, ent.Pos.Y, ent.Size.W, ent.Size.H})
			}
		}
		renderer.Present()
	}
}
