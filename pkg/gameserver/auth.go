package gameserver

import (
	"fmt"
	"log"

	"github.com/cfoust/sour/pkg/gameserver/protocol/cubecode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
)

func (s *Server) setAuthRole(client *Client, rol role.ID, domain, name string) {
	authUser := fmt.Sprintf("'%s'", cubecode.Magenta(name))
	if domain != "" {
		authUser = fmt.Sprintf("'%s' [%s]", cubecode.Magenta(name), cubecode.Green(domain))
	}

	if client.Role >= rol {
		msg := fmt.Sprintf("%s authenticated as %s", s.Clients.UniqueName(client), authUser)
		s.Clients.Message(msg)
		log.Println(cubecode.SanitizeString(msg))
	} else {
		msg := fmt.Sprintf("%s claimed %s privileges as %s", s.Clients.UniqueName(client), rol, authUser)
		s.Clients.Message(msg)
		log.Println(cubecode.SanitizeString(msg))
		s._setRole(client, rol)
	}
}

func (s *Server) setRole(client *Client, targetCN uint32, rol role.ID) {
	target := s.Clients.GetClientByCN(targetCN)
	if target == nil {
		client.Message(cubecode.Fail(fmt.Sprintf("no client with CN %d", targetCN)))
		return
	}
	if target.Role == rol {
		return
	}
	if client != target && client.Role <= target.Role || client == target && rol != role.None {
		client.Message(cubecode.Fail("you can't do that"))
		return
	}

	var msg string
	if rol == role.None {
		if client == target {
			msg = fmt.Sprintf("%s relinquished %s privileges", s.Clients.UniqueName(client), target.Role)
		} else {
			msg = fmt.Sprintf("%s took away %s privileges from %s", s.Clients.UniqueName(client), target.Role, s.Clients.UniqueName(target))
		}
	} else {
		msg = fmt.Sprintf("%s gave %s privileges to %s", s.Clients.UniqueName(client), rol, s.Clients.UniqueName(target))
	}
	s.Clients.Message(msg)
	log.Println(cubecode.SanitizeString(msg))

	s._setRole(target, rol)
}

func (s *Server) _setRole(client *Client, rol role.ID) {
	client.Role = rol
	message, _ := s.PrivilegedUsersPacket()
	s.Clients.Broadcast(message)
}
