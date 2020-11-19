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
	connection, err = grpc.Dial("localhost:50051", grpc.WithInsecure())
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
	// DropBucketRequest := bouncer.DropBucketParams{Login: testLogin, Ip: testIP}
	subnetRequest := bouncer.Subnet{Subnet: testSubnet}

	authResponse, err := client.Authorization(ctx, &authRequest)
	require.Nil(t, err)
	require.True(t, authResponse.GetOk())

	_, err = client.AddWhiteList(ctx, &subnetRequest)
	require.Nil(t, err)

}
