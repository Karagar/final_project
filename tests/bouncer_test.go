package bouncer

import (
	"log"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	bouncer "github.com/Karagar/final_project/pkg"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var err error
var ctx context.Context
var connection *grpc.ClientConn
var client bouncer.BouncerClient

func TestRun(t *testing.T) {
	connection, err = grpc.Dial("bouncer:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer connection.Close()
	ctx = context.Background()
	client = bouncer.NewBouncerClient(connection)

	unixNano := int(time.Now().UTC().UnixNano())
	testInt := rand.Intn(unixNano)
	testLogin := strconv.Itoa(testInt)
	testIP := strconv.Itoa(testInt%(rand.Intn(254)+1)) + "." + strconv.Itoa(testInt%(rand.Intn(254)+1)) + "."
	testIP = testIP + strconv.Itoa(testInt%(rand.Intn(254)+1)) + "." + strconv.Itoa(testInt%(rand.Intn(254)+1))
	testSubnet := testIP + "/24"
	testPassword := strconv.Itoa(rand.Intn(unixNano))
	authRequest := bouncer.AuthRequest{Login: testLogin, Ip: testIP, Password: testPassword}
	dropBucketRequest := bouncer.DropBucketParams{Login: testLogin, Ip: testIP}
	subnetRequest := bouncer.Subnet{Subnet: testSubnet}

	for i := 0; i < 10; i++ {
		authResponse, err := client.Authorization(ctx, &authRequest)
		require.Nil(t, err)
		require.True(t, authResponse.GetOk())
	}
	// Проверка переполнения
	authResponse, err := client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.False(t, authResponse.GetOk())

	//Провека white list
	_, err = client.AddWhiteList(ctx, &subnetRequest)
	require.Nil(t, err)

	// Проверка ответа
	authResponse, err = client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.True(t, authResponse.GetOk())

	//Провека удаления из white list
	_, err = client.RemoveWhiteList(ctx, &subnetRequest)
	require.Nil(t, err)

	time.Sleep(2 * time.Second)
	// Проверка ответа
	authResponse, err = client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.False(t, authResponse.GetOk())

	// Проверка очистки
	_, err = client.DropBucket(ctx, &dropBucketRequest)
	require.Nil(t, err)

	// Проверка ответа
	authResponse, err = client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.True(t, authResponse.GetOk())

	//Провека black list
	_, err = client.AddBlackList(ctx, &subnetRequest)
	require.Nil(t, err)

	// Проверка ответа
	authResponse, err = client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.False(t, authResponse.GetOk())

	//Провека удаления из black list
	_, err = client.RemoveBlackList(ctx, &subnetRequest)
	require.Nil(t, err)

	// Проверка ответа
	authResponse, err = client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.True(t, authResponse.GetOk())
}
