package client

import (
	"fmt"
	"github.com/lizuoqiang/gojsonrpc/common"
	"io"
	"net"
	"strconv"
	"time"
)

type Tcp struct {
	Ip          string
	Port        string
	RequestList []*common.SingleRequest
	Options     TcpOptions
	Conn        net.Conn
}

type TcpOptions struct {
	PackageEof       string
	PackageMaxLength int32
	ReadTimeOut      int
}

func NewTcpClient(ip string, port string) (*Tcp, error) {
	options := TcpOptions{
		"\r\n",
		1024 * 1024 * 2,
		15,
	}
	var addr = fmt.Sprintf("%s:%s", ip, port)
	conn, err := net.Dial("tcp", addr)
	//conn, err := net.DialTimeout("tcp", addr, time.Second*10)
	if err != nil {
		return nil, err
	}
	return &Tcp{
		ip,
		port,
		nil,
		options,
		conn,
	}, err
}

func (p *Tcp) BatchAppend(method string, params interface{}, result interface{}, isNotify bool) *error {
	singleRequest := &common.SingleRequest{
		method,
		params,
		result,
		new(error),
		isNotify,
	}
	p.RequestList = append(p.RequestList, singleRequest)
	return singleRequest.Error
}

func (p *Tcp) BatchCall() error {
	var (
		err error
		br  []interface{}
	)
	for _, v := range p.RequestList {
		var (
			req interface{}
		)
		if v.IsNotify == true {
			req = common.Rs(nil, v.Method, v.Params)
		} else {
			req = common.Rs(strconv.FormatInt(time.Now().Unix(), 10), v.Method, v.Params)
		}
		br = append(br, req)
	}
	bReq := common.JsonBatchRs(br)
	bReq = append(bReq, []byte(p.Options.PackageEof)...)
	err = p.handleFunc(bReq, p.RequestList)
	p.RequestList = make([]*common.SingleRequest, 0)
	return err
}

func (p *Tcp) SetOptions(tcpOptions interface{}) {
	p.Options = tcpOptions.(TcpOptions)
}

func (p *Tcp) Call(method string, params interface{}, result interface{}, isNotify bool) error {
	var (
		err error
		req []byte
	)
	if isNotify {
		req = common.JsonRs(nil, method, params)
	} else {
		req = common.JsonRs(strconv.FormatInt(time.Now().Unix(), 10), method, params)
	}
	req = append(req, []byte(p.Options.PackageEof)...)
	err = p.handleFunc(req, result)
	return err
}

func (p *Tcp) handleFunc(b []byte, result interface{}) error {
	var err error
	_, err = p.Conn.Write(b)
	if err != nil {
		return err
	}
	//var buf = make([]byte, p.Options.PackageMaxLength)
	//p.Conn.SetReadDeadline(time.Now().Add(p.Options.ReadTimeOut * time.Second))
	//n, err := p.Conn.Read(buf)
	//fmt.Println("buf:", string(buf), ",n:", n, ",err:", err)
	//if err != nil {
	//	return err
	//}
	//l := len([]byte(p.Options.PackageEof))
	//buf = buf[:n-l]

	var buf = make([]byte, p.Options.PackageMaxLength)
	var tmp = make([]byte, 1024)
	for {
		n, err := p.Conn.Read(tmp)
		fmt.Println("read: ", string(tmp), ",err:", err)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		buf = append(buf, tmp[:n]...)
	}
	//todo
	fmt.Println("buf1:", string(buf))
	eofLen := len([]byte(p.Options.PackageEof))
	bufLen := len(buf)
	buf = buf[:bufLen-eofLen]
	//todo
	fmt.Println("buf2:", string(buf), len(string(buf)))
	err = common.GetResult(buf, result)
	return err
}
