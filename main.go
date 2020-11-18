package main

import (
	bouncer "../pkg"
)

func main() {
	service := &bouncer.Service{}
	service.InitService()
}
