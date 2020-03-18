package defaultrpc

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	size := 10
	urls := "nats://localhost:4222,nats://localhost:5222,nats://localhost:6222"
	pool, err := New(urls, size)
	if err != nil {
		fmt.Println("Create Pool Error", err.Error())
	}
	var wg sync.WaitGroup

	//start listeners
	for i := 0; i < size/2; i++ {

		subject := "test_" + strconv.Itoa(i)

		wg.Add(1)
		go func(subject string) {
			defer wg.Done()

			conn, err := pool.Get()
			if err != nil {
				fmt.Println("Get Pool Error", err.Error())
			}
			defer pool.Put(conn)

			subscription, err := conn.SubscribeSync(subject)
			if err != nil {
				fmt.Println("SubscribeSync Error", err.Error())
			}
			defer subscription.Unsubscribe()

			msg, err := subscription.NextMsg(20 * time.Second)
			if err != nil {
				fmt.Println("Subscribe NextMsg Error", err.Error())
			}
			fmt.Println("Subscribe NextMsg : ", msg)

		}(subject)
	}

	time.Sleep(1 * time.Second)

	//start publishers
	for i := 0; i < size/2; i++ {

		subject := "test_" + strconv.Itoa(i)

		wg.Add(1)
		go func(subject string) {

			defer wg.Done()

			conn, err := pool.Get()
			if err != nil {
				fmt.Println("Get Pool Error", err.Error())
			}
			defer pool.Put(conn)

			err = conn.Publish(subject, []byte("hello"))

		}(subject)
	}

	wg.Wait()

	pool.Empty()
}

func TestConnection(t *testing.T) {
	urls := "nats://localhost:4222,nats://localhost:5222,nats://localhost:6222"
	pool, err := New(urls, 10)
	if err != nil {
		fmt.Println("Create Pool Error", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, err := pool.Get()
		if err != nil {
			fmt.Println("Get Pool Error", err.Error())
		}
		defer pool.Put(conn)

		subscription, err := conn.SubscribeSync("test_1")
		if err != nil {
			fmt.Println("SubscribeSync Error", err.Error())
		}
		defer subscription.Unsubscribe()

		msg, err := subscription.NextMsg(20 * time.Second)
		if err != nil {
			fmt.Println("Subscribe NextMsg Error", err.Error())
		}
		fmt.Println("Subscribe NextMsg : ", msg)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, err := pool.Get()
		if err != nil {
			fmt.Println("Get Pool Error", err.Error())
		}
		defer pool.Put(conn)

		err = conn.Publish("test_1", []byte("hello"))
	}()
}
