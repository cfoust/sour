package game

type Vector struct {
	X float64
	Y float64
	Z float64
}

type PhysicsState struct {
	State        byte
	LifeSequence int
	Move         int
	Yaw          int
	Roll         int
	Pitch        int
	Strafe       int
	O            Vector
	Falling      Vector
	Velocity     Vector
}
