package bouncer

import (
	"fmt"
	"os"
	"testing"
	"time"

	bouncer "github.com/Karagar/final_project/pkg"
)

var service *bouncer.Service

func init() {
	go func() {
		os.Setenv("CONFIG_FILE", "../config/config.json")
		service := &bouncer.Service{}
		service.InitService()
	}()
	fmt.Println("Wait for 1 sec untill service up")
	time.Sleep(1 * time.Second)
}

func TestRun(t *testing.T) {

	fmt.Println("Start")
	time.Sleep(12 * time.Second)
	fmt.Println("Finish")

}
