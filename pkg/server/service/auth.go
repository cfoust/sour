package service

import (
	"context"
)

func (server *Cluster) GreetClient(ctx context.Context, user *User) {
	server.NotifyClientChange(ctx, user, true)

	server.Users.Mutex.RLock()
	message := "users online: "
	users := server.Users.Users
	for i, other := range users {
		if other == user {
			message += "you"
		} else {
			message += other.Reference()
		}
		if i != len(users)-1 {
			message += ", "
		}
	}
	server.Users.Mutex.RUnlock()
	user.Message(message)
}
