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

	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

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
	WhiteList      []net.IPNet
	BlackList      []net.IPNet
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
	s.loadConfig()
	s.bucketBunch = map[string]buckets{}
	for k := range s.config.Limit {
		s.bucketBunch[k] = buckets{}
	}
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
	cfgFile := os.Getenv("CONFIG_FILE")
	if cfgFile == "" {
		cfgFile = "./config/config.json"
	}

	configJSONFile, err := os.Open(cfgFile)
	PanicOnErr(err)
	configByteValue, err := ioutil.ReadAll(configJSONFile)
	PanicOnErr(err)
	configJSONFile.Close()

	PanicOnErr(json.Unmarshal(configByteValue, &config))
	s.config = config
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
	if curBucket.FlagToDelition == true {
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

	for _, v := range s.config.WhiteList {
		if v.Contains(updatedIP) {
			isAlive = true
			needCheck = false
		}
	}

	if needCheck {
		for _, v := range s.config.BlackList {
			if v.Contains(updatedIP) {
				isAlive = false
				needCheck = false
			}
		}
	}

	return isAlive, needCheck
}

func (s *Service) Authorization(ctx context.Context, in *AuthRequest) (*AuthResponse, error) {
	log.Printf("new auth receive (Login=%v, Password=%v, Ip=%v)", in.Login, in.Password, in.Ip)

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
	s.RemoveWhiteList(ctx, in)
	_, updatedSubnet, err := net.ParseCIDR(in.Subnet)
	s.lock.Lock()
	s.config.BlackList = append(s.config.BlackList, *updatedSubnet)
	s.lock.Unlock()

	return &emptypb.Empty{}, err
}

func (s *Service) RemoveBlackList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	indexToRemove := -1
	for i, v := range s.config.BlackList {
		if v.String() == in.Subnet {
			indexToRemove = i
		}
	}
	if indexToRemove >= 0 {
		s.lock.Lock()
		s.config.BlackList = append(s.config.BlackList[:indexToRemove], s.config.BlackList[indexToRemove+1:]...)
		s.lock.Unlock()
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) AddWhiteList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	s.RemoveBlackList(ctx, in)
	_, updatedSubnet, err := net.ParseCIDR(in.Subnet)
	s.lock.Lock()
	s.config.WhiteList = append(s.config.WhiteList, *updatedSubnet)
	s.lock.Unlock()

	return &emptypb.Empty{}, err
}

func (s *Service) RemoveWhiteList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	indexToRemove := -1
	for i, v := range s.config.WhiteList {
		if v.String() == in.Subnet {
			indexToRemove = i
		}
	}
	if indexToRemove >= 0 {
		s.lock.Lock()
		s.config.WhiteList = append(s.config.WhiteList[:indexToRemove], s.config.WhiteList[indexToRemove+1:]...)
		s.lock.Unlock()
	}

	return &emptypb.Empty{}, nil
}

func PanicOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
