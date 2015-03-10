// Copyright (C) 2015 The Protogalaxy Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//go:generate protoc --go_out=plugins=grpc:. -I ../protos ../protos/gameroom.proto

package gameroom

import (
	"errors"
	"sync"

	"golang.org/x/net/context"

	"code.google.com/p/go-uuid/uuid"
)

type room struct {
	ID          string
	Owner       string
	OtherPlayer string
}

func (r *room) join(userID string) error {
	if r.OtherPlayer != "" {
		return ErrRoomFull
	}
	r.OtherPlayer = userID
	return nil
}

var ErrRoomFull = errors.New("room full")

type Generator interface {
	GenerateID() string
}

type UUIDGenerator struct{}

func (g *UUIDGenerator) GenerateID() string {
	return uuid.NewRandom().String()
}

type RoomManager struct {
	lock  sync.Mutex
	rooms map[string]*room

	Generator Generator
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:     make(map[string]*room),
		Generator: &UUIDGenerator{},
	}
}

func (m *RoomManager) CreateRoom(ctx context.Context, req *CreateRequest) (*CreateReply, error) {
	var rep CreateReply
	if m.isUserInAnyRoom(req.UserId) {
		rep.Status = ResponseStatus_ALREADY_IN_ROOM
		return &rep, nil
	}

	room := &room{
		ID:    m.Generator.GenerateID(),
		Owner: req.UserId,
	}
	m.lock.Lock()
	m.rooms[room.ID] = room
	m.lock.Unlock()

	rep.Status = ResponseStatus_SUCCESS
	rep.RoomId = room.ID
	return &rep, nil
}

func (m *RoomManager) RoomInfo(ctx context.Context, req *InfoRequest) (*InfoReply, error) {
	var rep InfoReply
	m.lock.Lock()
	defer m.lock.Unlock()

	room, ok := m.rooms[req.RoomId]
	if !ok {
		rep.Status = ResponseStatus_ROOM_NOT_FOUND
		return &rep, nil
	}

	rep.Status = ResponseStatus_SUCCESS
	rep.Room = &Room{
		Id: room.ID,
	}
	return &rep, nil
}

func (m *RoomManager) JoinRoom(ctx context.Context, req *JoinRequest) (*JoinReply, error) {
	var rep JoinReply
	m.lock.Lock()
	defer m.lock.Unlock()

	room, ok := m.rooms[req.RoomId]
	if !ok {
		rep.Status = ResponseStatus_ROOM_NOT_FOUND
		return &rep, nil
	}

	if m.isUserInAnyRoom(req.UserId) {
		rep.Status = ResponseStatus_ALREADY_IN_ROOM
		return &rep, nil
	}

	err := room.join(req.UserId)
	switch {
	case err == ErrRoomFull:
		rep.Status = ResponseStatus_ROOM_FULL
		return &rep, nil
	case err != nil:
		return nil, err
	}

	rep.Status = ResponseStatus_SUCCESS
	return &rep, nil
}

func (m *RoomManager) isUserInAnyRoom(userID string) bool {
	for _, room := range m.rooms {
		if userInRoom(room, userID) {
			return true
		}
	}
	return false
}

func userInRoom(r *room, userID string) bool {
	return r.Owner == userID || r.OtherPlayer == userID
}
