/**
 * @Time   : 2026/6/23 00:16
 * @Author : chenyangzhao542@gmail.com
 * @File   : generator.go
 **/

package id

type Generator interface {
	New() string
}
