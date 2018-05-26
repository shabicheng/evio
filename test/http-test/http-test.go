package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func request() {
	//生成client 参数为默认
	client := &http.Client{}

	//生成要访问的url
	url := "http://127.0.0.1:8088"

	//提交请求
	reqest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		panic(err)
	}
	for i := 0; i < 1000; i++ {
		//处理返回结果
		response, err := client.Do(reqest)
		if err != nil {
			fmt.Printf("error %v \n", err)
			continue
		}

		//将结果定位到标准输出 也可以直接打印出来 或者定位到其他地方进行相应的处理
		//stdout := os.Stdout
		//_, err = io.Copy(stdout, response.Body)
		//ioutil.ReadAll(response.Body)
		response.Body.Close()
		//返回的状态码
		//status := response.StatusCode

		//fmt.Println(status)
	}

}

func run(id int, wg *sync.WaitGroup) {
	startTime := time.Now()

	request()

	endTime := time.Now()
	fmt.Printf("done %d %v \n", id, endTime.Sub(startTime))
	wg.Done()
}

func main() {
	var wg sync.WaitGroup
	startTime := time.Now()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go run(i, &wg)
	}

	wg.Wait()
	endTime := time.Now()
	fmt.Printf("Total done  %v \n", endTime.Sub(startTime))
}
