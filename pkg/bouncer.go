package bouncer

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	sync "sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const defaultConfigPath = "../config/config.json"

type Service struct {
	lock        sync.RWMutex
	bucketBunch map[string]buckets
	config      ConfigStruct
	server      *grpc.Server
	listener    net.Listener
}

type ConfigStruct struct {
	ListenerAdress string
	TimerSec       int64
	Limit          map[string]int
	Lists          map[string][]net.IPNet
}

type buckets map[string]bucketDetail

type bucketDetail struct {
	MainChan       chan int64
	FlagToDelition bool
}

func (s *Service) InitService() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.loadConfig()
	s.initValues()
	s.InitRemover(ctx)

	lsn, err := net.Listen("tcp", s.config.ListenerAdress)
	PanicOnErr(err)
	log.Printf("Starting server on %s", lsn.Addr().String())

	s.listener = lsn
	s.server = grpc.NewServer()
	RegisterBouncerServer(s.server, s)
	err = s.server.Serve(lsn)
	PanicOnErr(err)
}

func (s *Service) ShutDown() {
	s.server.Stop()
	s.listener.Close()
}

func (s *Service) InitRemover(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.config.TimerSec) * time.Second)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.RemoveEmptyBuckets()
			}
		}
	}()
}

func (s *Service) loadConfig() {
	config := ConfigStruct{}
	cfgFile := os.Getenv("CONFIG_PATH")
	if cfgFile == "" {
		cfgFile = defaultConfigPath
	}

	configJSONFile, err := os.Open(cfgFile)
	PanicOnErr(err)
	configByteValue, err := ioutil.ReadAll(configJSONFile)
	PanicOnErr(err)
	configJSONFile.Close()

	PanicOnErr(json.Unmarshal(configByteValue, &config))
	s.config = config
}

func (s *Service) initValues() {
	s.bucketBunch = map[string]buckets{}
	for k := range s.config.Limit {
		s.bucketBunch[k] = buckets{}
	}
}

func (s *Service) RemoveEmptyBuckets() {
	s.lock.Lock()
	for bucketType, bucketsByType := range s.bucketBunch {
		for key, bucket := range bucketsByType {
			if bucket.FlagToDelition {
				close(bucket.MainChan)
				delete(s.bucketBunch[bucketType], key)
			}
			bucket.FlagToDelition = true
		}
	}
	s.lock.Unlock()
}

func (s *Service) RemoveBucket(bucketType string, bucketKey string) {
	s.lock.Lock()
	close(s.bucketBunch[bucketType][bucketKey].MainChan)
	delete(s.bucketBunch[bucketType], bucketKey)
	s.lock.Unlock()
}

func (s *Service) addToBucket(bucketType string, bucketKey string) (isAlive bool) {
	curBucket, ok := s.bucketBunch[bucketType][bucketKey]
	if !ok {
		curBucket = bucketDetail{
			MainChan:       make(chan int64, s.config.Limit[bucketType]),
			FlagToDelition: false,
		}
		s.lock.Lock()
		s.bucketBunch[bucketType][bucketKey] = curBucket
		s.lock.Unlock()
	}
	if curBucket.FlagToDelition {
		curBucket.FlagToDelition = false
		s.lock.Lock()
		s.bucketBunch[bucketType][bucketKey] = curBucket
		s.lock.Unlock()
	}

	curBucketChan := curBucket.MainChan
	now := time.Now().Unix()
	oldTime := time.Now().Unix() - s.config.TimerSec

	select {
	case curBucketChan <- now:
		return true
	default:
		for {
			nextElem := <-curBucketChan
			if nextElem > oldTime {
				break
			}
			if len(curBucketChan) == 0 {
				break
			}
		}
	}

	select {
	case curBucketChan <- now:
		return len(curBucketChan) < s.config.Limit[bucketType]
	default:
		return false
	}
}

func (s *Service) checkLists(address string) (isAlive bool, needCheck bool) {
	updatedIP := net.ParseIP(address)
	needCheck = true
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, v := range s.config.Lists["white"] {
		if v.Contains(updatedIP) {
			isAlive = true
			needCheck = false
		}
	}

	if needCheck {
		for _, v := range s.config.Lists["black"] {
			if v.Contains(updatedIP) {
				isAlive = false
				needCheck = false
			}
		}
	}

	return isAlive, needCheck
}

func (s *Service) Authorization(ctx context.Context, in *AuthRequest) (*AuthResponse, error) {
	isAlive, needCheck := s.checkLists(in.Ip)
	if needCheck {
		loginAnswer := s.addToBucket("login", in.Login)
		passwordAnswer := s.addToBucket("password", in.Password)
		ipAnswer := s.addToBucket("ip", in.Ip)
		isAlive = loginAnswer && passwordAnswer && ipAnswer
	}

	return &AuthResponse{Ok: isAlive}, nil
}

func (s *Service) DropBucket(ctx context.Context, in *DropBucketParams) (*emptypb.Empty, error) {
	s.RemoveBucket("ip", in.Ip)
	s.RemoveBucket("login", in.Login)

	return &emptypb.Empty{}, nil
}

func (s *Service) AddBlackList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.AddSubnetToList(in.Subnet, "black")
}

func (s *Service) RemoveBlackList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.RemoveSubnetFromList(in.Subnet, "black")
}

func (s *Service) AddWhiteList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.AddSubnetToList(in.Subnet, "white")
}

func (s *Service) RemoveWhiteList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.RemoveSubnetFromList(in.Subnet, "white")
}

func (s *Service) AddSubnetToList(subnet string, listType string) error {
	oppositeListType := "white"
	if oppositeListType == listType {
		oppositeListType = "black"
	}
	err := s.RemoveSubnetFromList(subnet, oppositeListType)
	if err != nil {
		return errors.Wrap(err, "Adding subnet to list")
	}

	_, updatedSubnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return errors.Wrap(err, "Adding subnet to list")
	}

	s.lock.Lock()
	s.config.Lists[listType] = append(s.config.Lists[listType], *updatedSubnet)
	s.lock.Unlock()

	return nil
}

func (s *Service) RemoveSubnetFromList(subnet string, listType string) error {
	indexToRemove := -1
	_, updatedSubnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return errors.Wrap(err, "Removing subnet from list")
	}
	subnet = updatedSubnet.String()

	for i, v := range s.config.Lists[listType] {
		if v.String() == subnet {
			indexToRemove = i
		}
	}
	if indexToRemove >= 0 {
		s.lock.Lock()
		s.config.Lists[listType] = append(s.config.Lists[listType][:indexToRemove], s.config.Lists[listType][indexToRemove+1:]...)
		s.lock.Unlock()
	}
	return nil
}

func PanicOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
