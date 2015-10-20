package main

import (
	"math"
	"time"
)

type GameScene struct {
	sch      SceneChannels
	lastTime time.Time
	keyState [1024]bool
	states   []*GameState
}

// load and run the scene. this is called inside a goroutine from the engine
func (s *GameScene) Load(sceneCh SceneChannels) {
	s.sch = sceneCh
	s.states = make([]*GameState, 2) // make 2 gamestates so we can look back one frame
	for i := range s.states {
		s.states[i] = &GameState{}
	}

	// load our assets
	s.sch.Eng <- EngineCommand{Id: EC_LOADIMAGE, Data: "base/player.png"}
	img := <-s.sch.Eng
	engImg := img.Data.(Image)

	// load our "level" here
	s.states[0].Entities[0] = Entity{Valid: true, Pos: Vector{100, 100}, Size: Size{64, 128}, Color: RGBA{255, 0, 0, 255}, Image: engImg.Id}

	// block on the first gamestate so we can sync with the renderer
	// FIXME: emit a loading screen immediately inside the load function
	s.sch.RCmd <- s.render(s.states[0])

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

		s.sch.RCmd <- s.render(s.states[0])
		s.lastTime = now
	}
}

func (s *GameScene) update(dt int32, userCmd UserCommand) {
	// shift our states down (this is somewhere where we should eventually not rely on GC)
	s.states[1] = s.states[0]
	s.states[0] = &GameState{}

	// copy the entities from the last state
	s.states[0].Entities = s.states[1].Entities

	st := s.states[0]
	st.FrameTime = dt
	st.Time = s.states[1].Time + dt/1000000

	for i := 0; i < len(st.Entities); i++ {
		var ent *Entity = &st.Entities[i]
		if !ent.Valid {
			continue
		}

		ent.Vel.X = 1 * (userCmd.Right - userCmd.Left) / 255
		ent.Vel.Y = 1 * (userCmd.Down - userCmd.Up) / 255

		// move all entities based on velocity
		ent.Pos.X += st.FrameTime / 5000000 * ent.Vel.X
		ent.Pos.Y += st.FrameTime / 5000000 * ent.Vel.Y

		// bounds checking
		if ent.Pos.X+ent.Size.W > 800 || ent.Pos.X < 0 {
			ent.Pos.X = BoundInt(ent.Pos.X, 0, 800-ent.Size.W)
		}

		if ent.Pos.Y+ent.Size.H > 600 || ent.Pos.Y < 0 {
			ent.Pos.Y = BoundInt(ent.Pos.Y, 0, 600-ent.Size.H)
		}

		// whee colors! not used since we have an image now
		ent.Color.G = (ent.Color.G + 1) % 255
	}
}

func (s *GameScene) render(st *GameState) *RenderCommandList {
	var commandList RenderCommandList
	num := 0

	for ; num < 450; num++ {
		var cmd RectCommand
		commandList.Commands[num].Id = RC_RECT
		cmd.Pos = Vector{(int32(2*num)+int32(float64(st.Time))/20)%832 - 32, int32(math.Sin(float64(st.Time)/3000+float64(num))*300) + 300}
		cmd.Size = Size{32, 32}
		cmd.Color = RGBA{0, 0, uint8(float64(cmd.Pos.Y)/600*200) + 55, 255}
		commandList.Commands[num].Data = cmd
	}

	for _, ent := range st.Entities {
		if !ent.Valid {
			continue
		}

		if ent.Image > -1 {
			var cmd PicCommand
			commandList.Commands[num].Id = RC_PIC
			cmd.Pos = ent.Pos
			cmd.Size = ent.Size
			cmd.SrcSize = Size{16, 32}
			// cmd.SrcPos
			commandList.Commands[num].Data = cmd
		} else {
			var cmd RectCommand
			commandList.Commands[num].Id = RC_RECT
			cmd.Pos = ent.Pos
			cmd.Size = ent.Size
			cmd.Color = ent.Color
			commandList.Commands[num].Data = cmd
		}

		num++
	}

	commandList.NumCommands = int32(num)

	return &commandList
}
