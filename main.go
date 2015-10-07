package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"runtime"
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
	ch := make(chan *GameState, 1)
	errCh := make(chan error)
	evCh := make(chan Event, 256)

	// load the gamescene and have it immediately start pumping out gamestates in a thread
	gameScene := GameScene{}
	go gameScene.Load(ch, evCh, errCh)

	// wait until the first frame before starting rendering
	var st *GameState = <-ch

	var running bool = true
	for running {

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				evCh <- Event{Type: EV_MOUSEMOVE, Position: Vector{t.X, t.Y}}

				// fmt.Printf("[%d ms] MouseMotion\ttype:%d\tid:%d\tx:%d\ty:%d\txrel:%d\tyrel:%d\n",
				// 	t.Timestamp, t.Type, t.Which, t.X, t.Y, t.XRel, t.YRel)

			case *sdl.MouseButtonEvent:
				evCh <- Event{Type: EV_MOUSECLICK, Down: t.State != 0}

				// fmt.Printf("[%d ms] MouseButton\ttype:%d\tid:%d\tx:%d\ty:%d\tbutton:%d\tstate:%d\n",
				// 	t.Timestamp, t.Type, t.Which, t.X, t.Y, t.Button, t.State)

			case *sdl.MouseWheelEvent:
				evCh <- Event{Type: EV_MOUSEWHEEL}

				// fmt.Printf("[%d ms] MouseWheel\ttype:%d\tid:%d\tx:%d\ty:%d\n",
				// 	t.Timestamp, t.Type, t.Which, t.X, t.Y)

			case *sdl.KeyDownEvent:
				evCh <- Event{Type: EV_KEY, Down: true, EvData1: int(t.Keysym.Scancode)}

				// fmt.Printf("[%d ms] Keyboard\ttype:%d\tsym:%c\tmodifiers:%d\tstate:%d\trepeat:%d\n",
				// t.Timestamp, t.Type, t.Keysym.Sym, t.Keysym.Mod, t.State, t.Repeat)

			case *sdl.KeyUpEvent:
				evCh <- Event{Type: EV_KEY, Down: false, EvData1: int(t.Keysym.Scancode)}

				// fmt.Printf("[%d ms] Keyboard\ttype:%d\tsym:%c\tmodifiers:%d\tstate:%d\trepeat:%d\n",
				// 	t.Timestamp, t.Type, t.Keysym.Sym, t.Keysym.Mod, t.State, t.Repeat)
			}
		}

		// check for new gamestates
		select {
		case st = <-ch: // we have a new gamestate
		case err = <-errCh: // we have an error from the gamestate
			return
		default:
		}

		// render whatever gamestate we have at the time
		renderer.Clear()
		// for now, all entities are just rectangles, so draw based on ent.pos and ent.size
		for _, ent := range st.Entities {
			if !ent.Valid {
				continue
			}

			renderer.SetDrawColor(ent.Color.R, ent.Color.G, ent.Color.B, ent.Color.A)
			renderer.FillRect(&sdl.Rect{ent.Pos.X, ent.Pos.Y, ent.Size.H, ent.Size.W})
		}
		renderer.Present()
		renderer.SetDrawColor(0, 0, 0, 255)
	}
}
