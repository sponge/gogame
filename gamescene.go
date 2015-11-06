package main

import (
	"fmt"
	"os"
	"time"

	"./tmx"
)

type GameScene struct {
	ready          bool
	sch            SceneChannels
	lastTime       time.Time
	keyState       [1024]bool
	prevState      GameState
	state          GameState
	renderingState GameState
	rcmds          RenderCommandList
	images         map[string]Image
	gmap           tmx.Map
}

// load and run the scene. this is called inside a goroutine from the engine
func (s *GameScene) Load(sceneCh SceneChannels) {
	s.sch = sceneCh
	s.images = make(map[string]Image)

	// load our assets
	playerImage := "player.png"
	s.sch.Eng <- EngineCommand{Id: EC_LOADIMAGE, Data: "base/" + playerImage}
	img := <-s.sch.Eng
	s.images[playerImage] = img.Data.(Image)

	// load our level here
	freader, err := os.Open("base/testlevel.tmx")
	if err != nil {
		return
	}

	gmap, err := tmx.Read(freader)
	if err != nil {
		return
	}
	s.gmap = *gmap

	for i := range s.gmap.Tilesets {
		ts := &s.gmap.Tilesets[i]
		s.sch.Eng <- EngineCommand{Id: EC_LOADIMAGE, Data: "base/" + ts.Image.Source}
		img := <-s.sch.Eng
		s.images[ts.Image.Source] = img.Data.(Image)
	}

	for i := range s.gmap.ObjectGroups {
		for j := range s.gmap.ObjectGroups[i].Objects {
			obj := s.gmap.ObjectGroups[i].Objects[j]
			fmt.Println(obj.Type)
		}
	}

	s.state.Entities[0] = Entity{Valid: true, Pos: Vector{100, 100}, Size: Size{64, 128}, Color: RGBA{255, 0, 0, 255}, Image: s.images["player.png"].Id}
	s.state.LocalEnt = &s.state.Entities[0]

	s.state.Camera.SetSize(Size{1280, 720})
	s.state.Camera.SetBounds(Size{int32(s.gmap.Width * 64), int32(s.gmap.Height * 64)})

	s.lastTime = time.Now()
	loop := time.Tick(8 * time.Millisecond)
	s.ready = true
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

		s.lastTime = now
	}
}

func (s *GameScene) update(dt int32, userCmd UserCommand) {
	s.sch.stateLock.Lock()
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
	}

	// move the camera so that the player is in the bounding box
	if st.LocalEnt.Vel.X > 0 && int(st.LocalEnt.Pos.X) > st.Camera.Right-200 {
		st.Camera.Set(int(st.LocalEnt.Pos.X-st.Camera.Size.W+200), st.Camera.Top)
	} else if st.LocalEnt.Vel.X < 0 && int(st.LocalEnt.Pos.X) < st.Camera.Left+200 {
		st.Camera.Set(int(st.LocalEnt.Pos.X-200), st.Camera.Top)
	}

	if st.LocalEnt.Vel.Y > 0 && int(st.LocalEnt.Pos.Y) > st.Camera.Bottom-200 {
		st.Camera.Set(st.Camera.Left, int(st.LocalEnt.Pos.Y-st.Camera.Size.H+200))
	} else if st.LocalEnt.Vel.Y < 0 && int(st.LocalEnt.Pos.Y) < st.Camera.Top+200 {
		st.Camera.Set(st.Camera.Left, int(st.LocalEnt.Pos.Y-200))
	}

	s.sch.stateLock.Unlock()
}

func (s *GameScene) render() *RenderCommandList {
	s.sch.stateLock.Lock()
	s.renderingState = s.state
	s.sch.stateLock.Unlock()

	s.rcmds = RenderCommandList{}
	st := &s.renderingState

	num := 0

	// for ; num < 1280/4; num++ {
	// 	var cmd RectCommand
	// 	s.rcmds.Commands[num].Id = RC_RECT
	// 	cmd.Pos = Vector{(int32(4*num)+int32(float64(st.Time))/20)%(1280+32) - 32, int32(math.Sin(float64(st.Time)/3000+float64(num))*(720/2)) + (720 / 2)}
	// 	cmd.Size = Size{32, 32}
	// 	cmd.Color = RGBA{0, 0, uint8(float64(cmd.Pos.Y)/720*200) + 55, 255}
	// 	s.rcmds.Commands[num].Data = cmd
	// }

	var cmd RectCommand
	s.rcmds.Commands[num].Id = RC_RECT
	cmd.Pos.X = 0
	cmd.Pos.Y = 0
	cmd.Size = st.Camera.Size
	cmd.Color = RGBA{168, 168, 168, 255}
	s.rcmds.Commands[num].Data = cmd
	num++

	var y, x, i, tid int
	maxX := min(st.Camera.Right/64+1, s.gmap.Width)
	maxY := min(st.Camera.Bottom/64+1, s.gmap.Height)

	for i = range s.gmap.Layers {
		layer := &s.gmap.Layers[i]
		tsw := layer.Tileset.Image.Width / layer.Tileset.TileWidth
		for y = max(0, st.Camera.Top/64); y < maxY; y++ {
			for x = max(0, st.Camera.Left/64); x < maxX; x++ {
				tid = int(layer.DecodedTiles[y*s.gmap.Width+x].ID)
				if tid == 0 {
					continue
				}

				var cmd PicCommand
				s.rcmds.Commands[num].Id = RC_PIC
				cmd.Pos = Vector{X: int32(x*64 - st.Camera.Left), Y: int32(y*64 - st.Camera.Top)}
				cmd.Size = Size{W: 64, H: 64}
				cmd.SrcSize = Size{int32(layer.Tileset.TileWidth), int32(layer.Tileset.TileHeight)}
				cmd.SrcPos = Vector{int32(tid%tsw) * int32(layer.Tileset.TileWidth), int32(tid/tsw) * int32(layer.Tileset.TileHeight)}
				cmd.ImageId = int32(s.images[layer.Tileset.Image.Source].Id)
				s.rcmds.Commands[num].Data = cmd
				num++
			}
		}

	}

	for _, ent := range st.Entities {
		if !ent.Valid {
			continue
		}

		if ent.Image > -1 {
			var cmd PicCommand
			s.rcmds.Commands[num].Id = RC_PIC
			cmd.Pos.X = ent.Pos.X - int32(st.Camera.Left)
			cmd.Pos.Y = ent.Pos.Y - int32(st.Camera.Top)
			cmd.Size = ent.Size
			cmd.SrcSize = Size{16, 32}
			// cmd.SrcPos
			s.rcmds.Commands[num].Data = cmd
		} else {
			var cmd RectCommand
			s.rcmds.Commands[num].Id = RC_RECT
			cmd.Pos.X = ent.Pos.X - int32(st.Camera.Left)
			cmd.Pos.Y = ent.Pos.Y - int32(st.Camera.Top)
			cmd.Size = ent.Size
			cmd.Color = ent.Color
			s.rcmds.Commands[num].Data = cmd
		}

		num++
	}

	s.rcmds.NumCommands = int32(num)
	return &s.rcmds
}
