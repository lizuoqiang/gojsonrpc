package client

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/lizuoqiang/gojsonrpc/common"
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
	TimeOut          int
}

func NewTcpClient(ip string, port string) (*Tcp, error) {
	options := TcpOptions{
		"\r\n",
		1024 * 1024 * 2,
		15,
	}
	var addr = fmt.Sprintf("%s:%s", ip, port)
	conn, err := net.Dial("tcp", addr)
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
	// 读写超时时间
	_ = p.Conn.SetDeadline(time.Now().Add(time.Duration(p.Options.TimeOut) * time.Second))

	// 传递参数
	var err error
	_, err = p.Conn.Write(b)
	if err != nil {
		return err
	}

	eof := []byte(p.Options.PackageEof)
	eofLen := len(eof)
	var (
		num  = 0
		data []byte
	)
	for {
		var buf = make([]byte, p.Options.PackageMaxLength)
		n, err := p.Conn.Read(buf)
		if err != nil {
			if n == 0 {
				return err
			}
			common.Debug(err.Error())
		}
		num += n
		data = append(data, buf[:n]...)
		if bytes.Equal(data[num-eofLen:], eof) {
			break
		}
	}
	err = common.GetResult(data[:num-eofLen], result)
	return err
}
