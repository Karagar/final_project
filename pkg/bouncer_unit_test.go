package bouncer

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var bouncer *Service

func init() {
	bouncer = &Service{}
	os.Setenv("CONFIG_PATH", "../config/config.json")
	bouncer.loadConfig()
	bouncer.initValues()
	fmt.Println("init Done")
}

func TestRun(t *testing.T) {
	unixNano := int(time.Now().UTC().UnixNano())
	testInt := rand.Intn(unixNano)
	testLogin := strconv.Itoa(testInt)
	testIP := strconv.Itoa(testInt%(rand.Intn(254)+1)) + "." + strconv.Itoa(testInt%(rand.Intn(254)+1)) + "."
	testIP = testIP + strconv.Itoa(testInt%(rand.Intn(254)+1)) + "." + strconv.Itoa(testInt%(rand.Intn(254)+1))
	testSubnet := testIP + "/24"
	var err error

	for i := 0; i <= bouncer.config.Limit["login"]; i++ {
		bouncer.addToBucket("login", testLogin)
	}
	// Проверка переполнения
	bouncer.addToBucket("login", testLogin)
	require.Equal(t, bouncer.config.Limit["login"], len(bouncer.bucketBunch["login"][testLogin].MainChan))

	// Проверка ответа
	target := bouncer.addToBucket("login", testLogin)
	require.False(t, target)

	// Проверка очистки
	bouncer.RemoveBucket("login", testLogin)
	bouncer.addToBucket("login", testLogin)
	require.Equal(t, 1, len(bouncer.bucketBunch["login"][testLogin].MainChan))

	// Проверка ответа
	target = bouncer.addToBucket("login", testLogin)
	require.True(t, target)

	//Провека black list
	target = bouncer.addToBucket("ip", testSubnet)
	require.True(t, target)
	err = bouncer.AddSubnetToList(testSubnet, "black")
	require.Nil(t, err)
	isAlive, needCheck := bouncer.checkLists(testIP)
	require.False(t, isAlive)
	require.False(t, needCheck)

	//Провека white list
	err = bouncer.AddSubnetToList(testSubnet, "white")
	require.Nil(t, err)
	isAlive, needCheck = bouncer.checkLists(testIP)
	require.True(t, isAlive)
	require.False(t, needCheck)
}
