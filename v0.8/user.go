package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C:    make(chan string),
		conn: conn,

		server: server,
	}

	// 启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

func (this *User) Online() {

	// 用户上线，将用户加入到onlineMap中
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()

	// 广播当前用户上限消息
	this.server.BroadCast(this, "已上线")
}

func (this *User) Offline() {
	// 用户下线，将用户从onlineMap中删除
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	// 广播当前用户上限消息
	this.server.BroadCast(this, "下线")
}

// 用户处理消息
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户
		fmt.Println(" 查询当前在线用户")
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + " online\n"
			this.sendMessage(onlineMsg)
		}
		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		fmt.Println(" 重命名")
		// 消息格式 rename|张三
		newName := strings.Split(msg, "|")[1]

		// 判断新名称是否被占用
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.sendMessage("当前用户名被使用\n")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()

			this.Name = newName
			this.sendMessage("您已经更新用户名:" + newName + "\n")
		}

	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式 to|张三|消息内容
		fmt.Println(" 私聊")
		// 1. 获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.sendMessage("消息格式不横琴，请使用 to|张三|消息内容 格式\n")
			return
		}

		// 2. 根据用户名，得到对方user对象
		remoteUser, ok := this.server.OnlineMap[remoteName]
		if !ok {
			this.sendMessage("该用户名不存在")
			return
		}
		// 3. 获取消息内容，通过对方usre对象将消息内容发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.sendMessage("无消息内容，请重发\n")
			return
		}
		remoteUser.sendMessage(this.Name + "对您说：" + content)
	} else {
		fmt.Println(" 什么都没有")
		this.server.BroadCast(this, msg)
	}
}

// 给当前user对应的客户端发送消息
func (this *User) sendMessage(msg string) {
	this.conn.Write([]byte(msg))
}

// 监听当前user channel的方法，一旦有消息，就直接发送给对应的客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C
		this.conn.Write([]byte(msg + "\n"))
	}
}
