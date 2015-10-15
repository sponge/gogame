package main

type Entity struct {
	Valid bool
	Pos   Vector
	Vel   Vector
	Size  Size
	Color RGBA
	Image int
}

type Vector struct {
	X int32
	Y int32
}

type Size struct {
	W int32
	H int32
}

type RGBA struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

type GameState struct {
	Time      int32
	FrameTime int32
	Entities  [1024]Entity
}

type EventType int

const (
	EV_KEY EventType = 1 + iota
	EV_MOUSEMOVE
	EV_MOUSECLICK
	EV_MOUSEWHEEL
)

type Event struct {
	Type     EventType
	Down     bool
	Position Vector
	EvData1  int
	EvData2  int
}

type Scene interface {
	Load(sceneCh SceneChannels)
	Unload()
}

type UserCommand struct {
	Up    int32
	Down  int32
	Left  int32
	Right int32
}

type SceneChannels struct {
	Gs  chan *GameState
	Ev  chan Event
	Eng chan EngineCommand
	Err chan error
}

type ECmd int

const (
	EC_UPLOADIMAGE ECmd = 1 + iota
)

type EngineCommand struct {
	Id      ECmd
	Success bool
	Data    interface{}
}

type Image struct {
	Id int
	W  int32
	H  int32
}
