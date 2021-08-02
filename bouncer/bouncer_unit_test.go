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
	fmt.Println("init Done")
}

func TestBouncer(t *testing.T) {
	unixNano := int(time.Now().UTC().UnixNano())
	testInt := rand.Intn(unixNano)
	testLogin := strconv.Itoa(testInt)
	testIP := strconv.Itoa(testInt%(rand.Intn(254)+1)) + "." + strconv.Itoa(testInt%(rand.Intn(254)+1)) + "."
	testIP = testIP + strconv.Itoa(testInt%(rand.Intn(254)+1)) + "." + strconv.Itoa(testInt%(rand.Intn(254)+1))
	testSubnet := testIP + "/24"
	var err error

	t.Run("bucket overflow", func(t *testing.T) {
		bouncer.initValues()
		for i := 0; i <= bouncer.config.Limit["login"]; i++ {
			bouncer.addToBucket("login", testLogin)
		}
		target := bouncer.addToBucket("login", testLogin)
		require.Equal(t, bouncer.config.Limit["login"], len(bouncer.bucketBunch["login"][testLogin].MainChan))
		require.False(t, target)
	})

	t.Run("bucket removing", func(t *testing.T) {
		bouncer.initValues()
		for i := 0; i <= bouncer.config.Limit["login"]; i++ {
			bouncer.addToBucket("login", testLogin)
		}
		bouncer.RemoveBucket("login", testLogin)
		target := bouncer.addToBucket("login", testLogin)
		require.Equal(t, 1, len(bouncer.bucketBunch["login"][testLogin].MainChan))
		require.True(t, target)
	})

	t.Run("whitelist", func(t *testing.T) {
		bouncer.initValues()
		target := true
		for i := 0; i <= bouncer.config.Limit["ip"]; i++ {
			target = bouncer.addToBucket("ip", testSubnet)
		}
		require.False(t, target)

		err = bouncer.AddSubnetToList(testSubnet, "white")
		require.Nil(t, err)
		isAlive, needCheck := bouncer.checkLists(testIP)
		require.True(t, isAlive)
		require.False(t, needCheck)
	})

	t.Run("blacklist", func(t *testing.T) {
		bouncer.initValues()
		target := bouncer.addToBucket("ip", testSubnet)
		require.True(t, target)

		err = bouncer.AddSubnetToList(testSubnet, "black")
		require.Nil(t, err)
		isAlive, needCheck := bouncer.checkLists(testIP)
		require.False(t, isAlive)
		require.False(t, needCheck)
	})
}
