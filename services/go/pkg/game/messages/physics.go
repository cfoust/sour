package messages

type Vector struct {
	X int
	Y int
	Z int
}

type PhysicsState struct {
	State        byte
	LifeSequence int
	Move         int
	Yaw          int
	Roll         int
	Pitch        int
	Strafe       int
	Strafe       int
	O            Vector
	Falling      Vector
	Velocity     Vector
}
