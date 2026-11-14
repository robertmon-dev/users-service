package utils

import (
	"users-service/app/models/ent"

	"github.com/golang-jwt/jwt/v5"
)

type PermissionInfo struct {
	Name   string `json:"name"`
	View   bool   `json:"view"`
	Add    bool   `json:"add"`
	Edit   bool   `json:"edit"`
	Delete bool   `json:"delete"`
}

type UserRoleInfo struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Permissions []PermissionInfo `json:"permissions"`
}

type Claims struct {
	UserID    int            `json:"user_id"`
	DeviceID  int            `json:"device_id"`
	UserRoles []UserRoleInfo `json:"user_roles"`
	jwt.RegisteredClaims
}

type JWTValidationResult struct {
	Claims *Claims `json:"claims"`
	Err    error   `json:"error"`
}

type JWTResult struct {
	User  *ent.User `json:"user"`
	Token string    `json:"token"`
	Err   error     `json:"error"`
}

type HashResult struct {
	Valid bool  `json:"valid"`
	Err   error `json:"error"`
}
