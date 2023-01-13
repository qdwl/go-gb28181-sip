package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"github.com/ghettovoice/gosip/transport"
	"github.com/qdwl/go-gb28181-sip/auth"
	"github.com/qdwl/go-gb28181-sip/stack"
	"github.com/qdwl/go-gb28181-sip/ua"
	"github.com/qdwl/go-gb28181-sip/utils"
)

type ServerConfig struct {
	ServerID          string    `json:"serverID"`          //服务器 id, 默认 34020000002000000001
	Realm             string    `json:"realm"`             //服务器域, 默认 3402000000
	ServerIp          string    `json:"serverIp"`          //服务器公网IP
	ServerPort        uint16    `json:"serverPort"`        //服务器公网端口
	UserName          string    `json:"userName"`          //服务器账号
	Password          string    `json:"password"`          //服务器密码
	UserAgent         string    `json:"userAgent"`         //服务器用户代理
	RegExpire         uint32    `json:"regExpire"`         //注册有效期，单位秒，默认 3600
	KeepaliveInterval uint32    `json:"keepaliveInterval"` //keepalive 心跳时间
	MaxKeepaliveRetry uint32    `json:"maxKeepaliveRetry"` //keeplive超时次数(超时之后发送重新发送reg)
	Transport         string    `json:"transport"`         //传输层协议(目前只支持udp,tcp)
	DisableAuth       bool      `json:"disableAuth"`       //是否启用鉴权
	LogLevel          log.Level `json:"logLevel"`          //日志级别
}

// Server
type Server struct {
	config *ServerConfig
	stack  *stack.SipStack
	ua     *ua.UserAgent
	log    log.Logger
}

// NewServer
func NewServer(config *ServerConfig) *Server {
	s := &Server{
		config: config,
		log:    utils.NewLogrusLogger(config.LogLevel, "Server", nil),
	}

	var authenticator *auth.ServerAuthorizer = nil

	if !config.DisableAuth {
		authenticator = auth.NewServerAuthorizer(s.requestCredential, config.Realm, false)
	}

	stack := stack.NewSipStack(&stack.SipStackConfig{
		UserAgent:  config.UserAgent,
		Extensions: []string{"replaces", "outbound"},
		Host:       config.ServerIp,
		ServerAuthManager: stack.ServerAuthManager{
			Authenticator:     authenticator,
			RequiresChallenge: s.requiresChallenge,
		},
		LogLevel: config.LogLevel,
	})

	stack.OnConnectionError(s.handleConnectionError)

	addr := fmt.Sprintf("0.0.0.0:%d", config.ServerPort)
	if err := stack.Listen("udp", addr); err != nil {
		s.Log().Panic(err)
	}

	if err := stack.Listen("tcp", addr); err != nil {
		s.Log().Panic(err)
	}

	ua := ua.NewUserAgent(&ua.UserAgentConfig{
		SipStack:  stack,
		UserName:  config.ServerID,
		Password:  config.Password,
		Realm:     config.Realm,
		Host:      config.ServerIp,
		LocalPort: config.ServerPort,
		Expires:   config.RegExpire,
		LogLevel:  config.LogLevel,
	})

	stack.OnRequest(sip.REGISTER, s.handleRegister)
	stack.OnRequest(sip.MESSAGE, s.handleMessage)

	s.stack = stack
	s.ua = ua

	return s
}

func (s *Server) Log() log.Logger {
	return s.log
}

// Shutdown .
func (s *Server) Shutdown() {
	s.ua.Shutdown()
}

func (s *Server) requestCredential(publicId string) (string, string, error) {
	return "", "", nil
}

func (s *Server) requiresChallenge(req sip.Request) bool {
	switch req.Method() {
	//case sip.UPDATE:
	case sip.REGISTER:
		return true
	case sip.INVITE:
		return true
	//case sip.RREFER:
	//	return false
	case sip.CANCEL:
		return false
	case sip.OPTIONS:
		return false
	case sip.INFO:
		return false
	case sip.BYE:
		{
			// Allow locally initiated dialogs
			// Return false if call-id in sessions.
			return false
		}
	}
	return false
}

func (s *Server) handleConnectionError(connError *transport.ConnectionError) {
	s.Log().Debugf("Handle Connection Lost: Source: %v, Dest: %v, Network: %v", connError.Source, connError.Dest, connError.Net)
	//b.registry.HandleConnectionError(connError)
}

func (s *Server) handleRegister(request sip.Request, tx sip.ServerTransaction) {
	resp := sip.NewResponseFromRequest(request.MessageID(), request, 200, "reason", "")
	sip.CopyHeaders("Expires", request, resp)
	utils.BuildContactHeader("Contact", request, resp, nil)
	tx.Respond(resp)
}

func (s *Server) handleMessage(request sip.Request, tx sip.ServerTransaction) {
	s.Log().Infof("handleMessage => %s, body => %s", request.Short(), request.Body())

	resp := sip.NewResponseFromRequest(request.MessageID(), request, 200, "", "")
	tx.Respond(resp)
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	serverConfig := &ServerConfig{
		ServerID:    "37021211002000000001",
		Realm:       "192.168.1.200",
		ServerIp:    "192.168.1.200",
		ServerPort:  7060,
		UserName:    "37021211002000000001",
		Password:    "12345678a",
		UserAgent:   "Go GB28181",
		DisableAuth: false,
		LogLevel:    log.DebugLevel,
	}

	server := NewServer(serverConfig)

	<-stop
	server.Shutdown()
}
