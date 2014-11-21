package main

import (
	"fmt"
	"os"

	"github.com/alytvynov/gcm-ccs"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: ./example project_number api_key registration_id")
		return
	}

	c, err := gcm.Dial(gcm.TestingAddr, os.Args[1], os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = c.Send(gcm.Message{
		To:   os.Args[3],
		Data: map[string]string{"omg": "friday!"},
	}); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("sent")

	r := <-c.Responses()
	fmt.Println("response:", r)

	c.Close()

	if c.Err() != nil {
		fmt.Println(c.Err())
	}

}
