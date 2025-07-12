package main

import (
	"fmt"
	"net"
	"strings"
)

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
	// 从 OnlineMap 中删除
	user.Server.mapLock.Lock()
	if _, exists := user.Server.OnlineMap[user.Name]; !exists {
		user.Server.mapLock.Unlock()
		return // 用户已经下线，直接返回
	}
	delete(user.Server.OnlineMap, user.Name)
	user.Server.mapLock.Unlock()

	// 广播用户下线消息
	user.Server.Broadcast(user, "已下线")

	// 关闭用户的连接
	if user.Conn != nil {
		err := user.Conn.Close()
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			fmt.Println("关闭连接失败: ", err)
		}
		user.Conn = nil
	}

	// 关闭用户的 channel（如果还未关闭）
	select {
	case <-user.Chan: // 如果 channel 已关闭，这里会立即返回
	default:
		close(user.Chan)
	}
}

// 给当前 user 对应的客户端发送消息
func (user *User) SendMsg(msg string) {
	if user.Conn == nil {
		return
	}
	user.Conn.Write([]byte(msg + "\n"))
}

// 用户处理消息的方法
func (user *User) DoMsg(msg string) {
	if msg == "who" {
		// 显示当前在线用户
		user.Server.mapLock.RLock()
		for _, cli := range user.Server.OnlineMap {
			onlineMsg := "[" + cli.Addr + "]" + cli.Name + "在线"
			user.SendMsg(onlineMsg)
		}
		user.Server.mapLock.RUnlock()
		return
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 修改用户名
		newName := msg[7:]

		// 检查新用户名是否已存在
		user.Server.mapLock.Lock()
		if _, exists := user.Server.OnlineMap[newName]; exists {
			user.SendMsg("用户名已存在，请重新命名")
			user.Server.mapLock.Unlock()
			return
		}

		// 修改用户名
		delete(user.Server.OnlineMap, user.Name)
		user.Name = newName
		user.Server.OnlineMap[newName] = user
		user.Server.mapLock.Unlock()
		user.SendMsg("用户名已修改为: " + newName)
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 私聊功能
		// 消息格式: "to|username|message"

		// 获取目标用户名和消息内容
		parts := strings.SplitN(msg[3:], "|", 2)
		if len(parts) < 2 {
			user.SendMsg("私聊格式错误，请使用: to|username|message")
			return
		}

		remoteName := parts[0]
		if remoteName == "" {
			user.SendMsg("私聊格式错误，请使用: to|username|message")
			return
		}

		remoteUser, ok := user.Server.OnlineMap[remoteName]
		if !ok {
			user.SendMsg("用户 " + remoteName + " 不在线或不存在")
			return
		}

		content := parts[1]
		if content == "" {
			user.SendMsg("私聊内容不能为空")
			return
		}
		remoteUser.SendMsg(user.Name + "对你说: " + content)
	} else {
		// 广播消息
		user.Server.Broadcast(user, msg)
	}
}

// 监听当前 user channel 的方法，一旦有消息，就直接发送给对端客户端
func (user *User) ListenMessage() {
	for {
		msg, ok := <-user.Chan
		if !ok {
			// channel已关闭，退出循环
			return
		}

		// 如果连接已关闭，退出循环
		if user.Conn == nil {
			return
		}

		// 发送消息，忽略错误（因为连接可能随时断开）
		user.Conn.Write([]byte(msg + "\n"))
	}
}
