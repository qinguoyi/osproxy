package base

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
)

// GetClientIp 获取本地网卡ip
func GetClientIp() (string, error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}

		}
	}

	return "", errors.New("can not find the client ip address")
}

// GetOutBoundIP 获取外网ip
func GetOutBoundIP() (string, error) {
	//向查询IP的网站发送GET请求
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	//读取响应的内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(body), nil

	//conn, err := net.Dial("udp", "8.8.8.8:53")
	//if err != nil {
	//	return "", err
	//}
	//localAddr := conn.LocalAddr().(*net.UDPAddr)
	//ip := strings.Split(localAddr.String(), ":")[0]
	//return ip, nil
}
