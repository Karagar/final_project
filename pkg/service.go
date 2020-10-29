package bouncer

import (
	"context"
	"log"
	"net"
	sync "sync"
	"time"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	lock        sync.RWMutex
	bucketBunch map[string]buckets
	config      ConfigStruct
}

type ConfigStruct struct {
	TimerSec  int64
	Limit     map[string]int
	WhiteList []net.IPNet
	BlackList []net.IPNet
}

type buckets map[string][]int64

func (s *Service) Init(ctx context.Context, config *ConfigStruct) {
	s.bucketBunch = map[string]buckets{}
	for k := range config.Limit {
		s.bucketBunch[k] = buckets{}
	}
	s.config = *config
	ticker := time.NewTicker(time.Duration(config.TimerSec) * time.Second)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()

				return
			case <-ticker.C:
				s.RemoveOldValues()
			}
		}
	}()
}

func (s *Service) RemoveOldValues() {
	tmp := map[string][]string{}
	now := time.Now().Unix()

	s.lock.RLock()
	for bucketType, bucketsByType := range s.bucketBunch {
		for key, bucket := range bucketsByType {
			if bucket[len(bucket)-1] < now-s.config.TimerSec {
				tmp[bucketType] = append(tmp[bucketType], key)
			}
		}
	}
	s.lock.RUnlock()
	s.RemoveBuckets(tmp)
}

func (s *Service) RemoveBuckets(toRemove map[string][]string) {
	s.lock.Lock()
	for bucketType, buckets := range toRemove {
		for _, bucket := range buckets {
			delete(s.bucketBunch[bucketType], bucket)
		}
	}
	s.lock.Unlock()
}

func (s *Service) addToBucket(bucketType string, bucketValue string) (isAlive bool) {
	s.lock.RLock()
	curBucket := s.bucketBunch[bucketType][bucketValue]
	s.lock.RUnlock()

	now := time.Now().Unix()

	curBucket = append(curBucket, now)
	// Проходим только по уже не актуальным датам
	for k, v := range curBucket {
		if v > now-s.config.TimerSec {
			curBucket = curBucket[k:]

			break
		}
	}

	s.lock.Lock()
	s.bucketBunch[bucketType][bucketValue] = curBucket
	s.lock.Unlock()

	return len(curBucket) < s.config.Limit[bucketType]
}

func (s *Service) checkLists(address net.IP) (isAlive bool, needCheck bool) {
	needCheck = true
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, v := range s.config.WhiteList {
		if v.Contains(address) {
			isAlive = true
			needCheck = false
		}
	}

	if needCheck {
		for _, v := range s.config.BlackList {
			if v.Contains(address) {
				isAlive = false
				needCheck = false
			}
		}
	}

	return isAlive, needCheck
}

func (s *Service) Authorization(ctx context.Context, in *AuthRequest) (*AuthResponse, error) {
	log.Printf("new auth receive (Login=%v, Password=%v, Ip=%v)",
		in.Login, in.Password, in.Ip)

	updatedIP := net.ParseIP(in.Ip)
	isAlive, needCheck := s.checkLists(updatedIP)
	if needCheck {
		loginAnswer := s.addToBucket("login", in.Login)
		passwordAnswer := s.addToBucket("password", in.Password)
		ipAnswer := s.addToBucket("ip", in.Ip)
		isAlive = loginAnswer && passwordAnswer && ipAnswer
	}

	return &AuthResponse{Ok: isAlive}, nil
}

func (s *Service) DropBucket(ctx context.Context, in *DropBucketParams) (*emptypb.Empty, error) {
	tmp := map[string][]string{}
	tmp["ip"] = append(tmp["ip"], in.Ip)
	tmp["login"] = append(tmp["login"], in.Login)
	s.RemoveBuckets(tmp)

	return &emptypb.Empty{}, nil
}

func (s *Service) AddBlackList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
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
