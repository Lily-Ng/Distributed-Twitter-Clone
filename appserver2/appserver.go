
package main

import (
	"fmt"
	// "net/http"
	// "io/ioutil"
	"bufio"
	"os"
	// "time"
	"log"
	"net"
	"sync"
	"strings"
)

// structure emulating a backend
var UserTable = [] User {}
var FollowsTable = [] Follows {}
var TweetTable = [] Tweet {}
var UserFile = sync.Mutex {}
var FollowsFile = sync.Mutex {}
var TweetFile = sync.Mutex {}

// structure that holds information relating to a user
type User struct{
	mut sync.Mutex
	Username string	//unlike DisplayName, this would be a primary key in database
	Password string
	DisplayName string
	Profile string
}

// structure holding information about each Tweet that exists
type Tweet struct{
	mut sync.Mutex
	author string	// references the username of the author
	message string
}

// show the relation that user1 follows user2
type Follows struct{
	mut sync.Mutex
	user1 string
	user2 string // both refers to username
}

// checks if a user exists on the site
func UserExists(name string) bool{
	for index, user := range UserTable{
		UserTable[index].mut.Lock()
		if user.Username == name{
			UserTable[index].mut.Unlock()
			return true
		}
		UserTable[index].mut.Unlock()
	}
	return false
}

// checks if user a follows user b
func afollowsb(user1, user2 string) bool{
	for index, entry := range FollowsTable{
		FollowsTable[index].mut.Lock()
		if entry.user1 == user1 && entry.user2 == user2 {
			FollowsTable[index].mut.Unlock()
			return true
		}
		FollowsTable[index].mut.Unlock()
	}
	return false
}

/*****************************************************************
*****************Functions to Update "Database"*******************
*****************************************************************/
// add a new user to the site
func CreateUser (new_username, new_password, new_disp_name, new_profile string, ch chan int) bool {
	if UserExists(new_username) == true {
		ch <- 1
		return false	// can't have duplicate username, but everything else can
	} else {
		UserTable = append(UserTable, User{sync.Mutex{}, new_username, new_password, new_disp_name, new_profile})
		UserFile.Lock()
		file, _ := os.OpenFile("UserTable.txt", os.O_APPEND|os.O_WRONLY, 0600)
		text := new_username + ";" + new_password + ";" + new_disp_name + ";" + new_profile + "\n"
		file.WriteString(text)
		file.Close()
		defer UserFile.Unlock()
		ch <- 1
		return true
	}
}

// remove a user entry from UserTable and all things related to that account
func DeleteUser (username string, ch chan int) {
	// obtain all necessary locks
	UserFile.Lock()
	FollowsFile.Lock()
	TweetFile.Lock()

	for UserIndex := 0; UserIndex < len(UserTable); UserIndex++ {
		if UserTable[UserIndex].Username == username {
			UserTable = append(UserTable[:UserIndex], UserTable[UserIndex+1:] ...)
		}
	}
	// clear out related entries in the FollowsTable
	for EntryIndex := len(FollowsTable)-1; EntryIndex >= 0; EntryIndex -- {
		fmt.Println(os.Stderr, EntryIndex)
		if FollowsTable[EntryIndex].user1 == username || FollowsTable[EntryIndex].user2 == username {
			FollowsTable = append(FollowsTable[:EntryIndex], FollowsTable[EntryIndex+1:] ...)
		}
	}
	// remove all tweets made by that user
	for TweetIndex := len(TweetTable)-1;  TweetIndex >= 0; TweetIndex -- {
		if TweetTable[TweetIndex].author == username {
			TweetTable = append(TweetTable[:TweetIndex], TweetTable[TweetIndex+1:] ...)
		}
	}
	UpdateAlltoFile()

	// release locks
	defer UserFile.Unlock()
	defer FollowsFile.Unlock()
	defer TweetFile.Unlock()
	ch <- 1
}

// used when user1 wants to follow user2 (insert entry into table)
func Follow (user1, user2 string, ch chan int) {
	FollowsTable = append(FollowsTable, Follows{sync.Mutex{}, user1, user2})
	FollowsFile.Lock()
	defer FollowsFile.Unlock()
	file, _ := os.OpenFile("FollowsTable.txt", os.O_APPEND|os.O_WRONLY, 0600)
	text := user1 + ";" + user2 + "\n"
	file.WriteString(text)
	file.Close()
	ch <- 1
}

// when user1 wants to unfollow user2. Unfollowing twice is the same as Unfollowing once.
func Unfollow (user1, user2 string, ch chan int) {
	FollowsFile.Lock()
	defer FollowsFile.Unlock()
	for EntryIndex :=0; EntryIndex < len(FollowsTable); EntryIndex ++ {
		if FollowsTable[EntryIndex].user1 == user1 && FollowsTable[EntryIndex].user2 == user2 {
			FollowsTable = append(FollowsTable[:EntryIndex], FollowsTable[EntryIndex+1:] ...)
			UpdateFollowsTable()
			ch <- 1
			return
		}
	}
	ch <- 1
}

// add a tweet to the TweetTable
func TweetThis(username, message string, ch chan int) {
	TweetFile.Lock()
	defer TweetFile.Unlock()
	TweetTable = append(TweetTable, Tweet{sync.Mutex{}, username, message})
	file, _ := os.OpenFile("TweetTable.txt", os.O_APPEND|os.O_WRONLY, 0600)
	text := username + ";" + message + "\n"
	file.WriteString(text)
	file.Close()
	ch <- 1
}

// send back an html follow/Unfollow button for a user
// option true = a "follow" button, option "false" = an unfollow button
func MakeFollowButton(username string, option bool) string {
	htmlcode, optionlabel := "", ""
	if option == true {
		htmlcode = "<form action=\"followuser\" method=\"post\">"
		optionlabel = "Follow"
	} else {
		htmlcode = "<form action=\"unfollowuser\" method=\"post\">"
		optionlabel = "Unfollow"
	}
	htmlcode += "<button type=\"submit\" name=\"user\" value=\"" + username + "\">"
	htmlcode += optionlabel
	htmlcode += "</button></form>"
	return htmlcode
}

// make a clickable link to the user(userhandle)'s profile.
func MakeProfileLink(userhandle string) string {
	urlcode := "<form id = \"gotoprofile\" action = \"usersearch\" method = \"post\">" + 
	"<input type = \"hidden\" name = \"targetuser\" value=\"" + userhandle + "\"></input>"
	// a button disguised as clickable text link
	urlcode += "<input type = \"submit\" style = \"background: none; cursor:pointer; padding:0px; border:none; font-size: 10px; text-decoration: none; color: black; \" value = \"@" + userhandle + "\"></input></form>"
	return urlcode
}

/*****************************************************************/

/*****************************************************************
******************Communication with Web Server"******************
*****************************************************************/
// send back the user table information
func SendBackUserTable(ch chan int) {

	service := "localhost:8081"

	conn, err := net.Dial("tcp", service)
	defer conn.Close()

	if err != nil {
		fmt.Fprint(os.Stderr, "Could not connect to web server", err.Error())
	}

	for _, user := range UserTable {
		fmt.Fprintf(conn, "%s;%s;%s;%s\r\n", user.Username, user.Password, user.DisplayName, user.Profile)
	}
	ch <- 1
}

// send back FollowsTable information
func SendBackFollowsTable(ch chan int) {

	service := "localhost:8081"

	conn, err := net.Dial("tcp", service)
	defer conn.Close()

	if err != nil {
		fmt.Fprint(os.Stderr, "Could not connect to web server", err.Error())
	}

	for _, followentry := range FollowsTable {
		fmt.Fprintf(conn, "%s;%s;\r\n", followentry.user1, followentry.user2)
	}
	ch <- 1
}

// send back the TweetTable information
func SendBackTweetTable(ch chan int) {

	service := "localhost:8081"

	conn, err := net.Dial("tcp", service)
	defer conn.Close()

	if err != nil {
		fmt.Fprint(os.Stderr, "Could not connect to web server", err.Error())
	}

	for _, tweetentry := range TweetTable {
		fmt.Fprintf(conn, "%s;%s\r\n", tweetentry.author, tweetentry.message)
	}
	ch <- 1
}

func SendBackAuthentication(loginok bool, ch chan int) {
	service := "localhost:8081"

	conn, err := net.Dial("tcp", service)
	defer conn.Close()

	if err != nil {
		fmt.Fprint(os.Stderr, "Could not connect to web server", err.Error())
	}

	if loginok == true {
		fmt.Fprintf(conn, "%s\r\n", "loginok")
	} else {
		fmt.Fprintf(conn, "%s\r\n", "wronglogin")
	}
	ch <- 1
}

// send back tweets by targetusername
func SendBackTweetsBy(targetusername string, ch chan int) {

	service := "localhost:8082"

	conn, err := net.Dial("tcp", service)
	defer conn.Close()

	if err != nil {
		fmt.Fprint(os.Stderr, "Could not connect to web server", err.Error())
	}

	tweetlist := GetTweetsBy(targetusername)

	for _, tweetentry := range tweetlist {
		fmt.Fprintf(conn, "%s;%s\r\n", tweetentry.author, tweetentry.message)
	}
	ch <- 1
}


/*****************************************************************/

/*****************************************************************
******************Functions to file with files"*******************
*****************************************************************/

// grab data from file for UserTable
func GetUserTable() {
	UserFile.Lock()
	defer UserFile.Unlock()
	file, _ := os.Open("UserTable.txt")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan(){
		line := scanner.Text()
		dataSlice := strings.Split(line, ";")
		UserTable = append(UserTable, User{sync.Mutex{}, dataSlice[0], dataSlice[1], dataSlice[2], dataSlice[3]})
	}
}

// backup userTable in UserTable.txt with current data
func UpdateUserTable() {
	os.Remove("UserTable.txt")
	file, _ := os.Create("UserTable.txt")
	defer file.Close()
	for _, thisuser := range UserTable {
		file.WriteString(thisuser.Username + ";" + thisuser.Password + ";" + thisuser.DisplayName + ";" + thisuser.Profile + "\n")
	}
}

// populate FollowsTable from file
func GetFollowsTable() {
	FollowsFile.Lock()
	defer FollowsFile.Unlock()
	file, _ := os.Open("FollowsTable.txt")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan(){
		line := scanner.Text()
		dataSlice := strings.Split(line, ";")
		FollowsTable = append(FollowsTable, Follows{sync.Mutex{}, dataSlice[0], dataSlice[1]})
	}
}

// backup FollowsTable in FollowsTable.txt with current data
func UpdateFollowsTable() {
	os.Remove("FollowsTable.txt")
	file, _ := os.Create("FollowsTable.txt")
	defer file.Close()
	for _, thisEntry := range FollowsTable {
		file.WriteString(thisEntry.user1 + ";" + thisEntry.user2 + "\n")
	}
}

// populate TweetTable from file
func GetTweetTable() {
	TweetFile.Lock()
	defer TweetFile.Unlock()
	file, _ := os.Open("TweetTable.txt")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan(){
		line := scanner.Text()
		dataSlice := strings.Split(line, ";")
		TweetTable = append(TweetTable, Tweet{sync.Mutex{}, dataSlice[0], dataSlice[1]})
	}
}

// backup TweetTable in TweetTable.txt with current data
func UpdateTweetTable() {
	os.Remove("TweetTable.txt")
	file, _ := os.Create("TweetTable.txt")
	defer file.Close()
	for _, thisTweet := range TweetTable {
		file.WriteString(thisTweet.author + ";" + thisTweet.message + "\n")
	}
}

// calls GetUserTable, GetFollowsTable, and GetTweetTable
func GetFromFile() {
	GetUserTable()
	GetFollowsTable()
	GetTweetTable()
}

// update all data to file on application server
func UpdateAlltoFile() {
	UpdateUserTable()
	UpdateFollowsTable()
	UpdateTweetTable()
}

/*****************************************************************/

/*****************************************************************
**************Functions to Retrieve from "Database"***************
*****************************************************************/
// verify user credentials
func Authenticate(tried_username, tried_password string) bool {
	for index, user := range UserTable {
		UserTable[index].mut.Lock()
		if user.Username == tried_username && user.Password == tried_password{
			UserTable[index].mut.Unlock()
			return true
		}
		UserTable[index].mut.Unlock()
	}
	return false
}

// obtain public information about a particular user (don't get password because that should be private)
func GetUserPublicInfo(username string) (string, string) {
	for index, user := range UserTable {
		UserTable[index].mut.Lock()
		if username == user.Username {
			UserTable[index].mut.Unlock()
			return user.DisplayName, user.Profile
		}
		UserTable[index].mut.Unlock()
	}
	return "",""
}

// obtain an array of people user is following
func GetFollowing(username string) [] string {
	FollowingList := [] string {}
	for index, entry := range FollowsTable{
		FollowsTable[index].mut.Lock()
		if entry.user1 == username{
			FollowingList = append(FollowingList, entry.user2)
		}
		FollowsTable[index].mut.Unlock()
	}
	return FollowingList
}

// obtain an array of people user is followed by
func GetFollowers(username string) [] string {
	FollowerList := [] string {}
	for index, entry := range FollowsTable{
		FollowsTable[index].mut.Lock()
		if entry.user2 == username{
			FollowerList = append(FollowerList, entry.user1)
		}
		FollowsTable[index].mut.Unlock()
	}
	return FollowerList
}

// get tweets posted by a user, sorted by most recent (start from the back)
func GetTweetsBy(username string) [] Tweet{
	tweetlist := [] Tweet {}
	for tweetindex := len(TweetTable)-1; tweetindex >= 0; tweetindex -- {
		TweetTable[tweetindex].mut.Lock()
		if TweetTable[tweetindex].author == username {
			tweetlist = append(tweetlist, TweetTable[tweetindex])
		}
		TweetTable[tweetindex].mut.Unlock()
	}
	return tweetlist
}

// get ten most recent tweets from people a user is following, to populate thwir Twitter feed
// note: the most recent tweets are stored towards the back of TweetTable, so we iterate it in reverse
func ObtainFeed(username string) [] Tweet {
	count := 0
	Feed := [] Tweet {}
	FollowingList := GetFollowing(username)
	for ind := len(TweetTable)-1; ind >= 0; ind-- {
	 	for _, following := range FollowingList{
	 		if TweetTable[ind].author == following && count <= 10{
	 			Feed = append(Feed, TweetTable[ind])
	 			count ++
	 		}
	 	}
	 	// the actual Twitter show your own tweets too in your feed
	 	if TweetTable[ind].author == username && count <= 10{
	 		Feed = append(Feed, TweetTable[ind])
	 			count ++
	 	}
	}
	return Feed
}

// return a pointer to the actual user entry in Usertable
func findUser(username string) *User{
	for userindex := 0; userindex <= len(UserTable); userindex++ {
		UserTable[userindex].mut.Lock()
		if UserTable[userindex].Username == username {
			UserTable[userindex].mut.Unlock()
			return &UserTable[userindex]
		}
		UserTable[userindex].mut.Unlock()
	}
	return nil
}

// update profile info on a user
// if pieces of info are not passed, it means that they will stay the same
func UpdateUserInfo(username, newpassword, newdispname, newprofile string, ch chan int) {
	thisuser := findUser(username)
	if thisuser == nil {
		log.Println("No such user.")
	}
	if newpassword != "" {
		thisuser.Password = newpassword
	}
	if newdispname != "" {
		thisuser.DisplayName = newdispname
	}
	if newprofile != "" {
		thisuser.Profile = newprofile
	}
	UpdateUserTable()
	ch <- 1
}

/*****************************************************************/

func main() {
	GetFromFile()	// prepare by pulling data from files

	ln, _ := net.Listen("tcp", ":8084")
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprint(os.Stderr, "Failed to accept")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "accept successful")
		scanner:= bufio.NewScanner(conn)
		var entry [][] string
		var parameters [] string
		var commandset bool = false
		var command string
		// the data received after the command will be parameters someone want to pass to the service
		for scanner.Scan() {
			if commandset == false{
				command = scanner.Text()
				commandset = true
			} else {
			line := scanner.Text()
			parameters = strings.Split(line, ";")
			entry = append(entry, parameters)
			}
		}

		fmt.Fprintln(os.Stderr, command)
		ch := make(chan int)
		switch command {
		case "ReqUserTable":
			go SendBackUserTable(ch)
		case "AddUser":
			for _, userentry := range entry {
				go CreateUser(userentry[0], userentry[1], userentry[2], userentry[3], ch)
			}
		case "DeleteUser":
			for _, userentry := range entry {
				go DeleteUser(userentry[0], ch)
			}
		case "UpdateUserInfo":
			for _, thisupdate := range entry {
				go UpdateUserInfo(thisupdate[0], thisupdate[1], thisupdate[2], thisupdate[3], ch)
			}
		case "ReqFollowsTable":
			go SendBackFollowsTable(ch)
		case "AddFollow":
			for _, followentry := range entry {
				go Follow(followentry[0], followentry[1], ch)
			}
		case "DeleteFollow":
			for _, followentry := range entry {
				go Unfollow(followentry[0], followentry[1], ch)
			}
		case "ReqTweetTable":
			go SendBackTweetTable(ch)
		case "AddTweet":
			for _, tweetentry := range entry {
				go TweetThis(tweetentry[0], tweetentry[1], ch)
			}
		case "GetTweetsBy":
			for _, target := range entry {
				go SendBackTweetsBy(target[0], ch)
			}
		case "AuthenticateUser":
			logincorrect:= Authenticate(entry[0][0], entry[0][1])
			go SendBackAuthentication(logincorrect, ch)
		default:
			ch <- 1
			fmt.Fprintln(os.Stderr, "invalid command")
		}
		<- ch
		fmt.Fprintln(os.Stderr, "connection gone!")
		conn.Close()
	}
}
