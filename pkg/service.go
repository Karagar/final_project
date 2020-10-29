package bouncer

import (
	"context"
	"fmt"
	"log"
	sync "sync"
	"time"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	lock        sync.RWMutex
	bucketBunch map[string]buckets
	timerSec    int64
	whiteList   []string
	blackList   []string
}

type buckets map[string][]int64

func (s *Service) Init(ctx context.Context, timerSec uint, whiteList []string, blackList []string) {
	s.bucketBunch = map[string]buckets{
		"login":    buckets{},
		"password": buckets{},
		"ip":       buckets{},
	}
	s.timerSec = int64(timerSec)
	s.whiteList = whiteList
	s.blackList = blackList
	ticker := time.NewTicker(time.Duration(timerSec) * time.Second)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case _ = <-ticker.C:
				s.RemoveOldValues()
			}
		}
	}()
	// TODO проверять по тикеру последнее значение каждого бакета, неактуальные удалять
	// TODO сделать инит из конфига
}

func (s *Service) RemoveOldValues() {
	tmp := map[string][]string{}
	now := time.Now().Unix()

	s.lock.RLock()
	for bucketType, bucketsByType := range s.bucketBunch {
		for key, bucket := range bucketsByType {
			if bucket[len(bucket)-1] < now-s.timerSec {
				tmp[bucketType] = append(tmp[bucketType], key)
			}
		}
	}
	s.lock.RUnlock()

	s.lock.Lock()
	for bucketType, buckets := range tmp {
		for _, bucket := range buckets {
			delete(s.bucketBunch[bucketType], bucket)
		}
	}
	s.lock.Unlock()
}

func (s *Service) addToBucket(bucketType string, bucketValue string) (isAlive bool, err error) {
	s.lock.RLock()
	curBucket := s.bucketBunch[bucketType][bucketValue]
	s.lock.RUnlock()

	now := time.Now().Unix()

	curBucket = append(curBucket, now)
	// Проходим только по уже не актуальным датам
	for k, v := range curBucket {
		if v > now-s.timerSec {
			curBucket = curBucket[k:]
			break
		}
	}

	s.lock.Lock()
	s.bucketBunch[bucketType][bucketValue] = curBucket
	s.lock.Unlock()

	return
}

func (s *Service) Authorization(ctx context.Context, in *AuthRequest) (*AuthResponse, error) {
	// TODO добавить конфиги для времени наблюдения, количества запросов, вайт/блек листов
	log.Printf("new auth receive (Login=%v, Password=%v, Ip=%v)",
		in.Login, in.Password, in.Ip)
	isAlive, err := s.addToBucket("login", in.Login)

	fmt.Println(s)

	return &AuthResponse{Ok: isAlive}, err
}

func (s *Service) DropBucket(ctx context.Context, in *DropBucketParams) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Service) AddBlackList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Service) RemoveBlackList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Service) AddWhiteList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Service) RemoveWhiteList(ctx context.Context, in *Subnet) (*emptypb.Empty, error) {
	return nil, nil
}
