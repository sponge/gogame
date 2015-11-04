package main

import "sync"

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
	Camera    Camera
	LocalEnt  *Entity
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
	RCmd      chan *RenderCommandList
	Ev        chan Event
	Eng       chan EngineCommand
	Err       chan error
	stateLock sync.Mutex
}

type ECmd int

const (
	EC_LOADIMAGE ECmd = 1 + iota
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

type RCmd int

const (
	RC_RECT RCmd = 1 + iota
	RC_PIC
	RC_TEXT
)

type RectCommand struct {
	Pos   Vector
	Size  Size
	Color RGBA
}

type PicCommand struct {
	ImageId int32
	Pos     Vector
	Size    Size
	SrcPos  Vector
	SrcSize Size
}

type RenderCommandList struct {
	NumCommands int32
	Commands    [4096]RenderCommand
}

type RenderCommand struct {
	Id   RCmd
	Data interface{}
}

type Camera struct {
	Left   int
	Right  int
	Top    int
	Bottom int
	Size   Size
}

func (s *Camera) Set(x int, y int) {
	s.Left = x
	s.Top = y
	s.Right = x + int(s.Size.W)
	s.Bottom = y + int(s.Size.H)
}

func (s *Camera) SetSize(sz Size) {
	s.Size = sz
	s.Set(s.Left, s.Top)
}
