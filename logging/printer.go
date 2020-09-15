package logging

import "fmt"

type Printer interface {
	Print(entry string, fields []LogField)
}

type consolePrinter struct {
	messageQueue chan string
}

func NewConsolePrinter() Printer {
	channel := make(chan string, 100)

	go func(ch chan string) {
		defer fmt.Println("printer has stopped!!!")

		for msg := range channel {
			fmt.Println(msg)
		}
	}(channel)

	return &consolePrinter{messageQueue: channel}
}

func (c *consolePrinter) Print(msg string, _ []LogField) {
	c.messageQueue <- msg
}
