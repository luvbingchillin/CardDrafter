package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		fmt.Printf("You added %s to the cli command", args[0])
	}
	fmt.Print("This to document/test learnings")
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter something random")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("An error:%s has occured", err)
		return
	}
	fmt.Printf("You inputed %s", input)
	// godotenv.Load() loads env data into os?
	if err := godotenv.Load(); err!=nil{
		return
	} 
	// os can now get env key pair values?
	dbUrl := os.Getenv("MONGODB_URL")
	// uses repository package to create a client/db object by passing in url
	client, db, err:= respository.ConnectDB(mongoURl, ....)
	//defers the disconnection once main finished to prevent conenctions from piling up
	defer clie......
	// uses build int net/http libirary to send a get reqeust to url
	resp, err:=http.get(url)
	//uses the json package to decosn response json data and place it in paylaod memeory, what is a decoder() ??
	if err := json.NewDecoder(resp.Body).Decode(&payload)...
	//uses mongodb library to assign a specific collection/table or db eqivalent?
	cardcolection := db.Collection("cards")
	//methods used by collection object to insert values, delte, create indexes et, need a context and either values of cust mongo predefined structs like indemodel
	.DeleteMany(ctx context.Context, values)
	.InsertMany
	.Indexes().CreateOne

}
