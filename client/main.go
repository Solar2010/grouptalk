package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

var (
	//退出管道命令
	chanQuit = make(chan bool,0)

	//服务端连接对象
	conn net.Conn
)
/*
错误处理
 */
func CHandleError(err error, why string)  {
	if err != nil {
		fmt.Println(why, err)
		os.Exit(1)
	}
}

func main() {
	nameInfo := [3]interface{}{"name","未阳","昵称"}

	retValuesMap := GetCmdLineArgs(nameInfo)

	name := retValuesMap["name"].(string)

	var e error
	conn, e = net.Dial("tcp", "127.0.0.1:8888")

	CHandleError(e, "net.Dial")

	defer func() {
		conn.Close()
	}()


	go handleSend(conn, name)

	go handleReceive(conn)

	<- chanQuit
}

/**
获取命令行参数
 */
func GetCmdLineArgs(argInfos ...[3]interface{}) (retValuesMap map[string]interface{})  {

	fmt.Printf("type=%T,value=%v", argInfos, argInfos)

	//初始化返回结果值

	retValuesMap = map[string]interface{}{}

	//预定义
	var strValuePtr *string

	var intValuePtr *int

	var strValuePtrsMap = map[string]*string{}
	var intValuePrtsMap = map[string]*int{}

	for _,argArray := range argInfos {

		nameValue := argArray[0].(string)

		usageValue := argArray[2].(string)

		switch argArray[1].(type) {
		case string:

			strValuePtr = flag.String(nameValue, argArray[1].(string), usageValue)

			strValuePtrsMap[nameValue] = strValuePtr
		case int:

			intValuePtr = flag.Int(nameValue, argArray[1].(int), usageValue)

			intValuePrtsMap[nameValue] = intValuePtr
		}
	}

	flag.Parse()

	if len(strValuePtrsMap) > 0 {
		for k, v := range strValuePtrsMap {
			retValuesMap[k] = *v
		}
	}

	if len(intValuePrtsMap) > 0 {
		for k, v := range intValuePrtsMap {
			retValuesMap[k] = *v
		}
	}
	return
}

/**
消息发送处理
 */
func handleSend(conn net.Conn, name string) {

	_, err := conn.Write([]byte(name))

	CHandleError(err, "conn.Write([]byte(name))")

	render := bufio.NewReader(os.Stdin)

	for {
		lineBytes, _, _ := render.ReadLine()
		lineStr := string(lineBytes)

		if strings.Index(lineStr, "upload") == 0 {
			strs := strings.Split(lineStr,"#")

			if len(strs) != 3 {
				fmt.Println("上传格式错误")
				continue
			}
			fileName := strs[1]
			filePath := strs[2]

			dataPack := make([]byte, 0)

			//写入数据包头
			header := make([]byte, 100)
			copy(header, []byte("upload#" +fileName+ "#"))
			dataPack = append(dataPack, header...)

			//写入数据包body
			fileBytes, _ := ioutil.ReadFile(filePath)
			dataPack = append(dataPack, fileBytes...)

			//发给服务端
			conn.Write(dataPack)
		} else {
			//发送到服务器
			_, err := conn.Write(lineBytes)
			CHandleError(err, "conn.Write(lineBytes)")

			if lineStr == "exit" {
				os.Exit(0)
			}
		}
	}
}
func handleReceive(conn net.Conn)  {

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)

		if err != io.EOF {
			CHandleError(err, "conn.Read")
		}

		if n > 0 {
			msg := string(buffer[:n])
			fmt.Println(msg)
		}
	}
}
