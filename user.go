package main

import "net"

type User struct {
	Name   string
	Addr   string
	Chan   chan string
	Conn   net.Conn
	Server *Server
}

// 创建一个用户的接口
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		Chan:   make(chan string),
		Conn:   conn,
		Server: server,
	}

	// 启动监听当前 user channel 消息的 goroutine
	go user.ListenMessage()

	return user
}

// 用户的上线业务
func (user *User) Online() {
	user.Server.mapLock.Lock()
	user.Server.OnlineMap[user.Name] = user
	user.Server.mapLock.Unlock()

	// 广播用户上线消息
	user.Server.Broadcast(user, "已上线")
}

// 用户的下线业务
func (user *User) Offline() {
	user.Server.mapLock.Lock()
	delete(user.Server.OnlineMap, user.Name)
	user.Server.mapLock.Unlock()

	// 广播用户下线消息
	user.Server.Broadcast(user, "已下线")
}

// 用户发送消息业务
func (user *User) SendMsg(msg string) {
	user.Server.Broadcast(user, msg)
}

// 监听当前 user channel 的方法，一旦有消息，就直接发送给对端客户端
func (user *User) ListenMessage() {
	for {
		msg := <-user.Chan
		user.Conn.Write([]byte(msg + "\n"))
	}
}
