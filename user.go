package main

import "net"

type User struct {
	Name string
	Addr string
	Chan chan string
	Conn net.Conn
}

// 创建一个用户的接口
func NewUser(conn net.Conn) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name: userAddr,
		Addr: userAddr,
		Chan: make(chan string),
		Conn: conn,
	}

	// 启动监听当前 user channel 消息的 goroutine
	go user.ListenMessage()

	return user
}

// 监听当前 user channel 的方法，一旦有消息，就直接发送给对端客户端
func (user *User) ListenMessage() {
	for {
		msg := <-user.Chan
		user.Conn.Write([]byte(msg + "\n"))
	}
}
