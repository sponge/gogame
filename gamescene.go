package main

import (
	//	"fmt"
	"time"
)

type GameScene struct {
	ch       chan *GameState
	evCh     chan Event
	errCh    chan error
	lastTime time.Time
	keyState [1024]bool
	states   []*GameState
}

func (s GameScene) Load(ch chan *GameState, evCh chan Event, errCh chan error) {
	s.ch = ch
	s.evCh = evCh
	s.errCh = errCh

	s.states = make([]*GameState, 2)
	for i := range s.states {
		s.states[i] = &GameState{}
	}

	s.states[0].Entities[0] = Entity{Valid: true, Pos: Vector{100, 100}, Size: Size{200, 200}, Color: RGBA{255, 0, 0, 255}}
	ch <- s.states[0]

	s.lastTime = time.Now()
	loop := time.Tick(5 * time.Millisecond)
	for now := range loop {
		dt := int32(time.Since(s.lastTime).Nanoseconds())

		// check for new inputs and generate a usercommand out of them
		var ev Event

		running := true
		for running {
			select {
			case ev = <-s.evCh: // we have a new event
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
		s.lastTime = now
	}
}

func (s GameScene) update(dt int32, userCmd UserCommand) {
	// shift our states down (this is somewhere where we should eventually not rely on GC)
	s.states[1] = s.states[0]
	s.states[0] = &GameState{}

	// keep the entities from the last state
	s.states[0].Entities = s.states[1].Entities

	st := s.states[0]
	st.FrameTime = dt

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

		ent.Color.G = (ent.Color.G + 1) % 255

		if ent.Pos.Y+ent.Size.H > 600 || ent.Pos.Y < 0 {
			ent.Pos.Y = BoundInt(ent.Pos.Y, 0, 600-ent.Size.H)
		}
	}

	// do a non blocking read on our gamestate channel to clear it if a previous state exists
	select {
	case _ = <-s.ch:
	default:
	}

	s.ch <- s.states[0]
}