package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	conn net.Conn
	name string
	addr string
}

type Group struct {
	Name string
	Owner *Client
	Members []*Client
}

var (

	allClientsMap = make(map[string]*Client)

	allGroupsMap map[string]*Group

	basePath = "~/xampp/htdocs/golang"
)

func init()  {
	allGroupsMap = make(map[string]*Group)
	allGroupsMap["示例群"] = NewGroup("示例群", &Client{name:"系统管理员"})
}

func SHandleError(err error, why string)  {
	if err != nil {
		fmt.Println(why, err)
		os.Exit(1)
	}
}

func main() {
	listener, e := net.Listen("tcp", "127.0.0.1:8888")
	SHandleError(e, "net.Listen")

	defer func() {
		for _, client := range allClientsMap {
			client.conn.Write([]byte("all:服务器进入维护状态，大家洗洗睡吧"))
		}
		listener.Close()
	}()

	conn, e := listener.Accept()
	SHandleError(e, "listener.Accept")
	clientAddr := conn.RemoteAddr()

	buffer := make([]byte, 1024)
	var clientName string

	for{
		n, err := conn.Read(buffer)
		SHandleError(err, "conn.Read")
		if n > 0 {
			clientName = string(buffer[:n])
			break
		}
	}
	fmt.Println(clientName + "上线了")

	client := &Client{conn, clientName, clientAddr.String()}
	allClientsMap[clientName] = client

	for _, client := range allClientsMap {
		client.conn.Write([]byte(clientName + "上线了"))
	}

	go ioWithClient(client);
}

func (g *Group) String() string {
	info := "群昵称:" + g.Name + "\n"
	info += "群主:" + g.Owner.name + "\n"
	info += "群人数:" + strconv.Itoa(len(g.Members)) + "\n"
	return info
}

func (g *Group) AddClient(client *Client)  {
	g.Members = append(g.Members, client)
}

func NewGroup(name string, owner *Client) *Group {
	group := new(Group)
	group.Name = name
	group.Owner = owner
	group.Members = make([]*Client, 0)
	group.Members = append(group.Members, owner)
	return  group
}

type GroupJoinReply struct {
	fromWhom *Client
	toWhom *Client
	group *Group
	answer string
}

func NewGroupJoinReply(fromWhom, toWhom *Client, group *Group, answer string) *GroupJoinReply {
	reply := new(GroupJoinReply)
	reply.fromWhom = fromWhom
	reply.toWhom = toWhom
	reply.group = group
	reply.answer = answer
	return  reply
}

func (reply *GroupJoinReply)AutoRun()  {
	if reply.group.Owner == reply.fromWhom {
		if reply.answer == "yes" {
			reply.group.AddClient(reply.toWhom)
			SendMessage2Client("你已经成功加入"+reply.group.Name, reply.toWhom)
		} else {
			SendMessage2Client(reply.group.Name+"群主已经拒绝了你的加群请求", reply.toWhom)
		}
	} else {
		SendMessage2Client("非法入群",reply.fromWhom)
	}
}
func ioWithClient(client *Client)  {

	buffer := make([]byte, 1024)

	for {
		n, err := client.conn.Read(buffer)

		if err != io.EOF {
			SHandleError(err, "client.conn.Read(buffer)")
		}

		if n > 0 {
			msgBytes := buffer[:n]

			//处理上传文件
			if bytes.Index(msgBytes, []byte("upload")) == 0 {

				//取数据包头
				msgStr := string(msgBytes[:100])
				fileName := strings.Split(msgStr,"#")[1]

				fileBytes := msgBytes[100:]

				err := ioutil.WriteFile(basePath + fileName, fileBytes, 0666)

				SHandleError(err, "ioutil.WriteFile")

				fmt.Println("文件上传成功")

				SendMessage2Client("文件上传成功",client)
			} else {
				//处理字符消息
				msg := string(msgBytes)
				fmt.Printf("%s:%s\n", client.name, msg)

				//聊天记录写文件缓存
				writeMsgToLog(msg, client)

				strs := strings.Split(msg, "#")

				if len(strs) > 1 {
					header := strs[0]
					body := strs[1]

					switch header {
					//全局消息
					case "all":
						handleWorldMsg(client, body)
						//建群
					case "grouP_steup":
						handleGroupStep(body, client)
						//查看群消息
					case "group_info":
						handleGroupInfo(body, client)
						//申请加群
					case "group_join":
						group, ok := allGroupsMap[body]

						if !ok {
							SendMessage2Client("群不存在", client)
							continue
						}

						SendMessage2Client(client.name +"申请加入群"+ body + "是否同意?", group.Owner)
						SendMessage2Client("申请已发送，请等待群主审核", client)
					case "group_joinreply":

						strs := strings.Split(body, "@")

						answer := strs[0]
						applicationName := strs[1]
						groupName := strs[2]

						//判断群昵称和申请人是否合法
						group, ok1 :=  allGroupsMap[groupName]
						toWhom, ok2 := allClientsMap[applicationName]

						if ok1 && ok2 {
							NewGroupJoinReply(client, toWhom, group, answer).AutoRun()
						}
					default:
						handleP2pMessage(header, client, body)
						
					}
				} else {
					//客户端主动下线
					if msg == "exit" {
						for name, c := range allClientsMap {
							if c == client {
								delete(allClientsMap, name)
							} else {
								c.conn.Write([]byte(name + "下线了"))
							}
						}
					} else if strings.Index(msg, "log@") == 0 {
						filterName := strings.Split(msg, "@")[1]
						go SendLog2Client(client,filterName)
					} else {
						client.conn.Write([]byte("已阅：" + msg))
					}
				}
			}
		}
	}
}

/*
消息写入文件
 */
func writeMsgToLog(msg string, client *Client)  {
	file, err := os.OpenFile("~/Xampp/htdocs/golang/log", os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0644)
	SHandleError(err, "os.OpenFile")

	defer file.Close()

	logMsg := fmt.Sprintln(time.Now().Format("2019-04-10 15:04:05"))

	file.Write([]byte(logMsg))
}
/**
处理群发消息
 */
func handleWorldMsg(client *Client, body string)  {
	for _, c := range allClientsMap {
		c.conn.Write([]byte(client.name + ":" + body))
	}
}

/**
建群操作
 */
func handleGroupStep(body string, client *Client) {
	if _, ok := allGroupsMap[body]; !ok {
		//建群
		newGroup := NewGroup(body, client)

		allGroupsMap[body] = newGroup

		SendMessage2Client("建群成功", client)

	} else {
		SendMessage2Client("群已存在", client)
	}
}

func handleGroupInfo(body string, client *Client)  {
	if(body == "all") {

		info := "";

		for _, group := range allGroupsMap {
			info += group.String() + "\n"
		}
		SendMessage2Client(info, client)
	} else {
		if group, ok := allGroupsMap[body]; ok {
			SendMessage2Client(group.String(), client)
		} else {
			SendMessage2Client("没有群信息", client)
		}
	}
}

func SendMessage2Client(msg string, client *Client)  {
	client.conn.Write([]byte(msg))
}

func handleP2pMessage(header string, client *Client, body string)  {
	for key, c := range allClientsMap {
		if key == header {
			c.conn.Write([]byte(client.name + ":" + body))

			go writeMessageToLog(client.name + ":" + body, c)
			break
		}
	}
}

/**
将每个人的聊天记录写入以他命名的log文件里
 */
func writeMessageToLog(msg string, client *Client)  {

	file, e := os.OpenFile("~/Xampp/htdocs/golang/logs/" + client.name + ".log", os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0644)

	SHandleError(e, "os.OpenFile")
	defer file.Close();

	logMsg := fmt.Sprintln(time.Now().Format("2019-04-12 15:04:05"), msg)

	file.Write([]byte(logMsg))
}

/**
返回客户端聊天日志
 */
func SendLog2Client(client *Client, filterName string)  {
	logBytes, e := ioutil.ReadFile("~/xampp/htdocs/golang/logs/" + client.name + ".log")
	SHandleError(e, "ioutil.ReadFile")

	if filterName != "all" {
		logStr := string(logBytes)
		targetStr := ""
		lineSlice := strings.Split(logStr, "\n")

		for _,lineStr := range lineSlice {
			if len(lineStr) > 20 {
				contentStr := lineStr[20:]

				if strings.Index(contentStr, filterName + "#") == 0 || strings.Index(contentStr, filterName + ":") == 0 {
					targetStr += lineStr + "\n"
				}
			}
		}
		client.conn.Write([]byte(targetStr))
	} else {
		client.conn.Write(logBytes)
	}
}