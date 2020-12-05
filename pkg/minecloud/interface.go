package minecloud

// World is a strong type for a world name.
type World string

// Interface is the main interface to Minecloud services
type Interface interface {
	Up(world World, instanceType *string) error
	Down(world World) error
}
