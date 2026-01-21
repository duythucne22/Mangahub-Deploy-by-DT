package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Common error codes - HTTP focused but protocol-aware
const (
	// HTTP Status Codes as strings for JSON responses
	ErrCodeValidation         = "VALIDATION_ERROR"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeUnauthorized       = "UNAUTHORIZED" 
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	
	// Protocol-specific error codes
	ErrCodeWebSocketClose     = "WEBSOCKET_CLOSE"
	ErrCodeUDPPacketInvalid   = "UDP_PACKET_INVALID"
	ErrCodeTCPFrameInvalid    = "TCP_FRAME_INVALID"
	ErrCodeGRPCServiceError   = "GRPC_SERVICE_ERROR"
)

// Common errors - SPEC.md compliant
var (
	// HTTP/Common errors
	ErrUserNotFound       = errors.New("user not found")
	ErrMangaNotFound      = errors.New("manga not found")
	ErrNotFound           = errors.New("resource not found")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameExists     = errors.New("username already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrForbidden          = errors.New("forbidden access")
	ErrInvalidInput       = errors.New("invalid input")
	
	// WebSocket protocol errors
	ErrWebSocketAuthFailed    = errors.New("websocket authentication failed")
	ErrWebSocketRoomNotFound  = errors.New("chat room not found")
	
	// UDP protocol errors
	ErrUDPPacketTooLarge      = errors.New("UDP packet exceeds 1KB limit")
	ErrUDPRateLimited         = errors.New("UDP rate limit exceeded (100 packets/second)")
	
	// TCP protocol errors  
	ErrTCPFrameTooLarge       = errors.New("TCP frame exceeds 1KB limit")
	ErrTCPInvalidMessageType  = errors.New("invalid TCP message type")
	
	// gRPC protocol errors
	ErrGRPCSearchFailed       = errors.New("gRPC search service unavailable")
)

// AppError - Enhanced for multi-protocol support
type AppError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Protocol    string                 `json:"protocol,omitempty"` // http, grpc, websocket, udp, tcp
	GRPCCode    codes.Code            `json:"grpc_code,omitempty"`
	WebSocketCode int                   `json:"websocket_code,omitempty"`
}

func (e *AppError) Error() string {
	if e.Protocol != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Protocol, e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ToHTTPError converts to HTTP-compatible error response
func (e *AppError) ToHTTPError() *APIResponse {
	return &APIResponse{
		Success: false,
		Error:   e.Message,
		Message: e.Message,
		Timestamp: time.Now(),
	}
}

// ToGRPCError converts to gRPC status error
func (e *AppError) ToGRPCError() error {
	return status.Error(e.GRPCCode, e.Message)
}

// ToWebSocketError returns WebSocket close code and message
func (e *AppError) ToWebSocketError() (int, string) {
	if e.WebSocketCode != 0 {
		return e.WebSocketCode, e.Message
	}
	
	// Default mapping
	switch e.Code {
	case ErrCodeUnauthorized:
		return websocket.ClosePolicyViolation, "authentication required"
	case ErrCodeForbidden:
		return websocket.ClosePolicyViolation, "forbidden access"
	case ErrCodeNotFound:
		return websocket.CloseNormalClosure, "resource not found"
	default:
		return websocket.CloseInternalServerErr, e.Message
	}
}

// ToUDPPacketError returns UDP error response format
func (e *AppError) ToUDPPacketError() []byte {
	return []byte(fmt.Sprintf("ERROR:%s:%s", e.Code, e.Message))
}

// Protocol-specific error constructors

// HTTP Errors
func NewHTTPError(code, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Protocol:   "http",
		Details:    map[string]interface{}{"original_error": err.Error()},
	}
}

// gRPC Errors  
func NewGRPCError(grpcCode codes.Code, code, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		GRPCCode:   grpcCode,
		Protocol:   "grpc",
		Details:    map[string]interface{}{"original_error": err.Error()},
	}
}

// WebSocket Errors
func NewWebSocketError(wsCode int, code, message string, err error) *AppError {
	return &AppError{
		Code:          code,
		Message:       message,
		WebSocketCode: wsCode,
		Protocol:      "websocket",
		Details:       map[string]interface{}{"original_error": err.Error()},
	}
}

// UDP Errors
func NewUDPError(code, message string, packetSize int) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Protocol:  "udp",
		Details:   map[string]interface{}{"packet_size": packetSize},
	}
}

// TCP Errors
func NewTCPError(code, message string, frameSize int) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Protocol:  "tcp",
		Details:   map[string]interface{}{"frame_size": frameSize},
	}
}