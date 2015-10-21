package main

import (
	"math"
	"time"

	"./gamemap"
)

type GameScene struct {
	sch       SceneChannels
	lastTime  time.Time
	keyState  [1024]bool
	prevState GameState
	state     GameState
	rcmds     RenderCommandList
}

// load and run the scene. this is called inside a goroutine from the engine
func (s *GameScene) Load(sceneCh SceneChannels) {
	s.sch = sceneCh

	// load our assets
	s.sch.Eng <- EngineCommand{Id: EC_LOADIMAGE, Data: "base/player.png"}
	img := <-s.sch.Eng
	engImg := img.Data.(Image)

	// load our "level" here
	gamemap.Load("base/level1.json")
	s.state.Entities[0] = Entity{Valid: true, Pos: Vector{100, 100}, Size: Size{64, 128}, Color: RGBA{255, 0, 0, 255}, Image: engImg.Id}

	// block on the first gamestate so we can sync with the renderer
	// FIXME: emit a loading screen immediately inside the load function
	s.render(&s.state)
	s.sch.RCmd <- &s.rcmds

	s.lastTime = time.Now()
	loop := time.Tick(5 * time.Millisecond)
	for now := range loop {
		dt := int32(time.Since(s.lastTime).Nanoseconds())

		// check for new inputs and generate a usercommand out of them
		var ev Event

		running := true
		for running {
			select {
			case ev = <-s.sch.Ev: // we have a new event
				switch ev.Type {
				case EV_KEY:
					s.keyState[ev.EvData1] = ev.Down
				}
			default:
				running = false
			}
		}

		userCmd := UserCommand{}
		if s.keyState[82] {
			userCmd.Up = 255
		}

		if s.keyState[79] {
			userCmd.Right = 255
		}

		if s.keyState[81] {
			userCmd.Down = 255
		}

		if s.keyState[80] {
			userCmd.Left = 255
		}

		s.update(dt, userCmd)

		// do a non blocking read on our render command channel to clear it if a previous list exists
		select {
		case _ = <-s.sch.RCmd:
		default:
		}

		s.render(&s.state)
		s.sch.RCmd <- &s.rcmds
		s.lastTime = now
	}
}

func (s *GameScene) update(dt int32, userCmd UserCommand) {
	s.prevState = s.state

	st := &s.state
	st.FrameTime = dt
	st.Time = s.prevState.Time + dt/1000000

	for i := 0; i < len(st.Entities); i++ {
		var ent *Entity = &st.Entities[i]
		if !ent.Valid {
			continue
		}

		ent.Vel.X = 4 * (userCmd.Right - userCmd.Left) / 255
		ent.Vel.Y = 4 * (userCmd.Down - userCmd.Up) / 255

		// move all entities based on velocity
		ent.Pos.X += st.FrameTime / 5000000 * ent.Vel.X
		ent.Pos.Y += st.FrameTime / 5000000 * ent.Vel.Y

		// bounds checking
		if ent.Pos.X+ent.Size.W > 1280 || ent.Pos.X < 0 {
			ent.Pos.X = BoundInt(ent.Pos.X, 0, 1280-ent.Size.W)
		}

		if ent.Pos.Y+ent.Size.H > 720 || ent.Pos.Y < 0 {
			ent.Pos.Y = BoundInt(ent.Pos.Y, 0, 720-ent.Size.H)
		}

		// whee colors! not used since we have an image now
		ent.Color.G = (ent.Color.G + 1) % 255
	}
}

func (s *GameScene) render(st *GameState) {
	s.sch.RCmdLock.Lock()

	// FIXME: this causes crashes, race conditions in the thread
	//s.rcmds = RenderCommandList{}
	num := 0

	for ; num < 1280/4; num++ {
		var cmd RectCommand
		s.rcmds.Commands[num].Id = RC_RECT
		cmd.Pos = Vector{(int32(4*num)+int32(float64(st.Time))/20)%(1280+32) - 32, int32(math.Sin(float64(st.Time)/3000+float64(num))*(720/2)) + (720 / 2)}
		cmd.Size = Size{32, 32}
		cmd.Color = RGBA{0, 0, uint8(float64(cmd.Pos.Y)/720*200) + 55, 255}
		s.rcmds.Commands[num].Data = cmd
	}

	for _, ent := range st.Entities {
		if !ent.Valid {
			continue
		}

		if ent.Image > -1 {
			var cmd PicCommand
			s.rcmds.Commands[num].Id = RC_PIC
			cmd.Pos = ent.Pos
			cmd.Size = ent.Size
			cmd.SrcSize = Size{16, 32}
			// cmd.SrcPos
			s.rcmds.Commands[num].Data = cmd
		} else {
			var cmd RectCommand
			s.rcmds.Commands[num].Id = RC_RECT
			cmd.Pos = ent.Pos
			cmd.Size = ent.Size
			cmd.Color = ent.Color
			s.rcmds.Commands[num].Data = cmd
		}

		num++
	}

	s.rcmds.NumCommands = int32(num)

	s.sch.RCmdLock.Unlock()

	return
}
