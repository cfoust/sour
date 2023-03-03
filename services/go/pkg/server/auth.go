package server

import (
	"fmt"
	"log"

	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/role"
)

func (s *Server) handleAuthRequest(client *Client, domain string, name string, onSuccess func(role.ID), onFailure func(error)) {
	sessionID := client.SessionID

	_onSuccess := func(rol role.ID) {
		s.callbacks <- func() {
			if client.SessionID != sessionID {
				return
			}
			if domain == "" {
				log.Printf("successful gauth by %s as '%s'", client.String(), name)
			} else {
				log.Printf("successful auth by %s as '%s' [%s]", client.String(), name, domain)
			}
			client.Authentications[domain].name = name
			onSuccess(rol)
		}
	}

	_onFailure := func(err error) {
		s.callbacks <- func() {
			if client.SessionID != sessionID {
				return
			}
			if domain == "" {
				log.Printf("unsuccessful gauth try by %s as '%s': %v", client.String(), name, err)
			} else {
				log.Printf("unsuccessful auth try by %s as '%s' [%s]: %v", client.String(), name, domain, err)
			}
			delete(client.Authentications, domain)
			onFailure(err)
		}
	}

	s.AuthManager.TryAuthentication(
		domain,
		name,
		func(reqID uint32, chal string) {
			s.callbacks <- func() {
				if client.SessionID != sessionID {
					return
				}
				client.Authentications[domain] = &Authentication{reqID: reqID}
				client.Send(nmc.AuthChallenge, domain, reqID, chal)
			}
		},
		_onSuccess,
		_onFailure,
	)
}

func (s *Server) handleAuthAnswer(client *Client, domain string, reqID uint32, answ string) {
	s.AuthManager.CheckAnswer(reqID, domain, answ)
}

func (s *Server) setAuthRole(client *Client, rol role.ID, domain, name string) {
	authUser := fmt.Sprintf("'%s'", cubecode.Magenta(name))
	if domain != "" {
		authUser = fmt.Sprintf("'%s' [%s]", cubecode.Magenta(name), cubecode.Green(domain))
	}

	if client.Role >= rol {
		msg := fmt.Sprintf("%s authenticated as %s", s.Clients.UniqueName(client), authUser)
		s.Clients.Broadcast(nmc.ServerMessage, msg)
		log.Println(cubecode.SanitizeString(msg))
	} else {
		msg := fmt.Sprintf("%s claimed %s privileges as %s", s.Clients.UniqueName(client), rol, authUser)
		s.Clients.Broadcast(nmc.ServerMessage, msg)
		log.Println(cubecode.SanitizeString(msg))
		s._setRole(client, rol)
	}
}

func (s *Server) setRole(client *Client, targetCN uint32, rol role.ID) {
	target := s.Clients.GetClientByCN(targetCN)
	if target == nil {
		client.Send(nmc.ServerMessage, cubecode.Fail(fmt.Sprintf("no client with CN %d", targetCN)))
		return
	}
	if target.Role == rol {
		return
	}
	if client != target && client.Role <= target.Role || client == target && rol != role.None {
		client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
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
	s.Clients.Broadcast(nmc.ServerMessage, msg)
	log.Println(cubecode.SanitizeString(msg))

	s._setRole(target, rol)
}

func (s *Server) _setRole(client *Client, rol role.ID) {
	client.Role = rol
	typ, pup, _ := s.PrivilegedUsersPacket()
	s.Clients.Broadcast(typ, pup)
}
