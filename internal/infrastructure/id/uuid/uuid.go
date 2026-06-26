/**
 * @Time   : 2026/6/23 00:18
 * @Author : chenyangzhao542@gmail.com
 * @File   : uuid.go
 **/

package uuid

import "github.com/google/uuid"

type UUIDv4Generator struct{}

func NewUUIDv4Generator() UUIDv4Generator {
	return UUIDv4Generator{}
}

func (UUIDv4Generator) New() string {
	return uuid.NewString()
}
