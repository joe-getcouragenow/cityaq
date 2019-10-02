package main

import "github.com/ctessum/cityaq/gui"

func main() {
	conn := gui.DefaultConnection()
	c := gui.NewCityAQ(conn)
	c.Monitor()

	select {} // Block
}
