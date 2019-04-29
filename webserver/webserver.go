
package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"bufio"
	"os"
	"time"
	"log"
	"net"
	"strings"
)

// clients have knowledge of every replica (what ports they use, in this case.)
// addresses of each replicas are stored as strings, as net.Dial takes in an address string
// here, we intend to follow the model of read-one write-all protocol
var AddressBook = [] string {"localhost:8083", "localhost:8084", "localhost:8085"}

// structure emulating a backend
// var UserTable = [] User {}
// var FollowsTable = [] Follows {}
// var TweetTable = [] Tweet {}

// structure that holds information relating to a user
type User struct{
	Username string	//unlike DisplayName, this would be a primary key in database
	Password string
	DisplayName string
	Profile string
}

// structure holding information about each Tweet that exists
type Tweet struct{
	author string	// references the username of the author
	message string
}

// show the relation that user1 follows user2
type Follows struct{
	user1 string
	user2 string // both refers to username
}

// checks if a user exists on the site
func UserExists(name string) bool{
	myusertable := RequestUserTable()
	for _, user := range myusertable{
		if user.Username == name{
			return true
		}
	}
	return false
}

// checks if user a follows user b
func afollowsb(user1, user2 string) bool{
	myfollowstable := RequestFollowsTable()
	for _, entry := range myfollowstable{
		if entry.user1 == user1 && entry.user2 == user2 {
			return true
		}
	}
	return false
}

/*****************************************************************
*****************Functions to Update "Database"*******************
*****************************************************************/
// add a new user to the site
func CreateUser (new_username, new_password, new_disp_name, new_profile string) bool {
	if UserExists(new_username) == true {
		return false	// can't have duplicate username, but everything else can
	} else {
		// sent request to every replica
		for _, address := range AddressBook {
			service := address
			conn, err := net.DialTimeout("tcp", service, time.Second)
			if err != nil {
				fmt.Fprint(os.Stderr, "could not connect\n")
			} else {
				fmt.Fprintf(conn, "AddUser\r\n")
				fmt.Fprintf(conn, "%s;%s;%s;%s",new_username, new_password, new_disp_name, new_profile)
				conn.Close()
			}
		}
		return true
	}
}

// remove a user entry from UserTable and all things related to that account
func DeleteUser (username string) {
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
		} else {
			fmt.Fprintf(conn, "DeleteUser\r\n")
			fmt.Fprintf(conn, "%s",username)
			conn.Close()
		}
	}
}

// used when user1 wants to follow user2 (insert entry into table)
func Follow (user1, user2 string) {
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
		} else {
			fmt.Fprintf(conn, "AddFollow\r\n")
			fmt.Fprintf(conn, "%s;%s",user1, user2)
			conn.Close()
		}
	}
}

// when user1 wants to unfollow user2. Unfollowing twice is the same as Unfollowing once.
func Unfollow (user1, user2 string) {
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
		} else {
			fmt.Fprintf(conn, "DeleteFollow\r\n")
			fmt.Fprintf(conn, "%s;%s",user1, user2)
			conn.Close()
		}
	}
}

// add a tweet to the TweetTable
func TweetThis(username, message string) {
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
		} else {
			fmt.Fprintf(conn, "AddTweet\r\n")
			fmt.Fprintf(conn, "%s;%s",username, message)
			conn.Close()
		}
	}
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
*****************Request Application Server"**********************
*****************************************************************/
// everything below are reads, so they only request from one replica
// get UserTable from appserver and return it
func RequestUserTable() [] User {
	var myusertable [] User
	// contact the first replica available, going down the AddressBook
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
			continue
		} else {
			fmt.Fprintf(conn, "%s\n","ReqUserTable")
			conn.Close()
			break
		}
	}

	ln, _ := net.Listen("tcp", ":8081")
	defer ln.Close()
	
	for {
		conn, _ := ln.Accept()
		scanner:= bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			data := strings.Split(line, ";")
			myusertable = append(myusertable, User{data[0], data[1], data[2], data[3]})
		}
		return myusertable
	}
}

// get FollowsTable from appserver and return it
func RequestFollowsTable() [] Follows {
	var myfollowstable [] Follows
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
			continue
		} else {
			fmt.Fprintf(conn, "%s\n","ReqFollowsTable")
			conn.Close()
			break
		}
		
	}
	

	ln, _ := net.Listen("tcp", ":8081")
	defer ln.Close()
	
	for {
		conn, _ := ln.Accept()
		scanner:= bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			data := strings.Split(line, ";")
			myfollowstable = append(myfollowstable, Follows{data[0], data[1]})
		}
		return myfollowstable
	}
}

// get TweetTable from appserver and return it
func RequestTweetTable() [] Tweet {
	var mytweettable [] Tweet
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
			continue
		} else {
			fmt.Fprintf(conn, "%s\n","ReqTweetTable")
			conn.Close()
			break
		}
	}
	

	ln, _ := net.Listen("tcp", ":8081")
	defer ln.Close()
	
	for {
		conn, _ := ln.Accept()
		scanner:= bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			data := strings.Split(line, ";")
			mytweettable = append(mytweettable, Tweet{data[0], data[1]})
		}
		return mytweettable
	}
}

/*****************************************************************/

/*****************************************************************
**************Functions to Retrieve from "Database"***************
*****************************************************************/
// verify user credentials
func Authenticate(tried_username, tried_password string) bool {
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
			continue
		} else {
			fmt.Fprintf(conn, "AuthenticateUser\n")
			fmt.Fprintf(conn, "%s;%s",tried_username, tried_password)
			conn.Close()
			break
		}
	}
	
	ln, _ := net.Listen("tcp", ":8081")
	defer ln.Close()
	
	for {
		conn, _ := ln.Accept()
		scanner:= bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "loginok"{
				return true
			} else {
				return false
			}
		}
	}
}

// obtain public information about a particular user (don't get password because that should be private)
func GetUserPublicInfo(username string) (string, string) {
	myusertable := RequestUserTable()
	for _, user := range myusertable {
		if username == user.Username {
			return user.DisplayName, user.Profile
		}
	}
	return "",""
}

// obtain an array of people user is following
func GetFollowing(username string) [] string {
	FollowingList := [] string {}
	myFollowingTable := RequestFollowsTable()
	for _, entry := range myFollowingTable{
		if entry.user1 == username{
			FollowingList = append(FollowingList, entry.user2)
		}
	}
	return FollowingList
}

// obtain an array of people user is followed by
func GetFollowers(username string) [] string {
	FollowerList := [] string {}
	myFollowingTable := RequestFollowsTable()
	for _, entry := range myFollowingTable{
		if entry.user2 == username{
			FollowerList = append(FollowerList, entry.user1)
		}
	}
	return FollowerList
}

// get tweets posted by a user, sorted by most recent (start from the back)
func GetTweetsBy(username string) [] Tweet{
	tweetlist := [] Tweet {}
	/*
	service := "localhost:8083"
	conn, err := net.Dial("tcp", service)
	if err != nil {
		fmt.Fprint(os.Stderr, "could not connect", err.Error())
	}
	fmt.Fprintf(conn, "GetTweetsBy\n")
	fmt.Fprintf(conn, "%s",username)
	conn.Close()
	ln, _ := net.Listen("tcp", ":8081")
	defer ln.Close()
	
	for {
		conn, _ := ln.Accept()
		scanner:= bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			data := strings.Split(line, ";")
			tweetlist = append(tweetlist, Tweet{data[0], data[1]})
		}
	}
	*/
	myTweetTable := RequestTweetTable()
	for tweetindex := len(myTweetTable)-1; tweetindex >= 0; tweetindex -- {
		if myTweetTable[tweetindex].author == username {
			tweetlist = append(tweetlist, myTweetTable[tweetindex])
		}
	}
	return tweetlist
}

// get ten most recent tweets from people a user is following, to populate their Twitter feed
// note: the most recent tweets are stored towards the back of TweetTable, so we iterate it in reverse
func ObtainFeed(username string) [] Tweet {
	count := 0
	Feed := [] Tweet {}
	FollowingList := GetFollowing(username)
	myTweetTable := RequestTweetTable()
	for ind := len(myTweetTable)-1; ind >= 0; ind-- {
	 	for _, following := range FollowingList{
	 		if myTweetTable[ind].author == following && count < 10{
	 			Feed = append(Feed, myTweetTable[ind])
	 			count ++
	 		}
	 	}
	 	// the actual Twitter show your own tweets too in your feed
	 	if myTweetTable[ind].author == username && count < 10{
	 		Feed = append(Feed, myTweetTable[ind])
	 			count ++
	 	}
	}
	return Feed
}

// return a pointer to the actual user entry in Usertable
func findUser(username string) *User{
	myusertable := RequestUserTable()
	for userindex := 0; userindex <= len(myusertable); userindex++ {
		if myusertable[userindex].Username == username {
			return &myusertable[userindex]
		}
	}
	return nil
}

// update profile info on a user
// if pieces of info are not passed, it means that they will stay the same
func UpdateUserInfo(username, newpassword, newdispname, newprofile string) {
	for _, address := range AddressBook {
		service := address
		conn, err := net.DialTimeout("tcp", service, time.Second)
		if err != nil {
			fmt.Fprint(os.Stderr, "could not connect\n")
		}
		fmt.Fprintf(conn, "UpdateUserInfo\n")
		fmt.Fprintf(conn, "%s;%s;%s;%s",username, newpassword, newdispname, newprofile)
		conn.Close()
	}
	
}

/*****************************************************************/

/******************************************************
********All handle functions go below this line********
******************************************************/
// tells the user that some required fields are missing when signing up
func AccountCreationFailed(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Account creation failed. Please fill out the required fields.")
}

func signup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
		LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
		// if user is not logged in, redirect them to /twitterclone
		if (err1 == nil && err2 == nil) {
			if Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == true {
				http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
			}
		}

		page, err := ioutil.ReadFile("./signup.html")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, string(page))
	case http.MethodPost:
		r.ParseForm()
		// make sure that all required fields are filled out before signing up new users
		if r.PostFormValue("new_login_name") != "" && r.PostFormValue("new_display_name") != "" && r.PostFormValue("new_acc_password") != "" {
			if strings.ContainsAny(r.PostFormValue("new_login_name"), ";") == true || strings.ContainsAny(r.PostFormValue("new_display_name"), ";") == true || strings.ContainsAny(r.PostFormValue("new_acc_password"), ";") == true || strings.ContainsAny(r.PostFormValue("new_profile"), ";") == true {
				// security policy: any of the fields may not contain semicolons
				fmt.Fprintf(w, "The fields may not contain semicolons.")
			} else if CreateUser(r.PostFormValue("new_login_name"), r.PostFormValue("new_acc_password"), r.PostFormValue("new_display_name"), r.PostFormValue("new_profile")) == true {
				fmt.Fprintf(w, "Welcome, %s! Your account had been created.", r.PostFormValue("new_display_name"))
			} else {
				fmt.Fprintf(w, "Username occupied. Please try a different one.")
			}
		} else {
			http.Redirect(w, r, "/AccountCreationFailed", http.StatusTemporaryRedirect)
		}
	}
}

// handle function for the main page "/twitterclone"
func login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		LOGGEDUSERNAME, err := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
		LOGGEDPASS, err := r.Cookie("LOGGEDPASS")
		if err != nil{
			log.Println(err)
		}
		// if user is not logged in, proceed as usual and take him/her to the normal login page
		if LOGGEDUSERNAME == nil || LOGGEDPASS == nil{
			page, err := ioutil.ReadFile("./login.html")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, string(page))
		} else {
			// if user is logged in, redirect to the logged in home page
			if Authenticate(LOGGEDUSERNAME.Value, LOGGEDUSERNAME.Value){
				http.Redirect(w, r, "/mainpage", http.StatusSeeOther)
			} else {
				fmt.Fprintf(w, "Incorrect credentials.")
			}
		}

	case http.MethodPost:
		r.ParseForm()
		if Authenticate(r.PostFormValue("login-name"), r.PostFormValue("login-password")){
			cookie1 := http.Cookie {
				Name: 		"LOGGEDUSERNAME",
				Value: 		r.PostFormValue("login-name"),
				Expires: 	time.Now().Add(1 * time.Hour),	// keep the user logged in for an hour
			}
			cookie2 := http.Cookie {
				Name: 		"LOGGEDPASS",
				Value: 		r.PostFormValue("login-password"),
				Expires: 	time.Now().Add(1 * time.Hour),	// keep the user logged in for an hour
			}
			http.SetCookie(w, &cookie1)
			http.SetCookie(w, &cookie2)
			http.Redirect(w, r, "/mainpage", http.StatusSeeOther)
		} else {
				fmt.Fprintf(w, "Incorrect credentials.")
		}
	}
}

func homescreen(w http.ResponseWriter, r *http.Request) {
	LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
	LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false{
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	page, err := ioutil.ReadFile("./home.html")
	if err != nil {
		log.Fatal(err)
	}
	var html string	// hold html content of actual feed
	var unfollowbutton string
	feed := ObtainFeed(LOGGEDUSERNAME.Value)
	for _, post := range feed {
		disp_name, _ := GetUserPublicInfo(post.author)
		if post.author != LOGGEDUSERNAME.Value {
			unfollowbutton = MakeFollowButton(post.author, false)
		}
		linktoprofile := MakeProfileLink(post.author)
		html += "<div style=\"text-decoration: underline;\" >" + disp_name + "</div>" + linktoprofile + unfollowbutton + "<br/>"
		html += "<div class=\"msg\" style=\"padding-left: 20px; font-size: 12px;\">" + post.message + "</div><br/>"
	}
	fmt.Fprintf(w, string(page) + html)
}

func signout(w http.ResponseWriter, r *http.Request) {
	_, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
	_, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil{
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	cookie1 := http.Cookie {
		Name: 		"LOGGEDUSERNAME",
		Value: 		"",
		Expires: 	time.Now(),	// cookie expired
	}
	cookie2 := http.Cookie {
		Name: 		"LOGGEDPASS",
		Value: 		"",
		Expires: 	time.Now(),	// cookie expired
	}
	http.SetCookie(w, &cookie1)
	http.SetCookie(w, &cookie2)
	http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
}

func tweet(w http.ResponseWriter, r *http.Request) {
	LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
	LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil{
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	message := r.PostFormValue("tweet_msg")
	if Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) && message != ""{
		if strings.ContainsAny(message, ";") {
			fmt.Fprintf(w, "You may not tweet anything containing semicolons.")
		} else {
			TweetThis(LOGGEDUSERNAME.Value, message)
		}
	}
	http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
}

func changeprofile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
		LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
		if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false{
			http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
		}
		page, err := ioutil.ReadFile("./requestchange.html")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, string(page))
	case http.MethodPost:
		LOGGEDUSERNAME, _ := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
		UpdateUserInfo(LOGGEDUSERNAME.Value, r.PostFormValue("new_password"), r.PostFormValue("new_display_name"), r.PostFormValue("new_profile"))
		if r.PostFormValue("new_password") != ""{
			cookie := http.Cookie {
			Name: 		"LOGGEDPASS",
			Value: 		r.PostFormValue("new_password"),
			Expires: 	time.Now().Add(1 * time.Hour),	// cookie expired
			}
			http.SetCookie(w, &cookie)
		}
		http.Redirect(w, r, "mainpage", http.StatusSeeOther)
	}
}

func deleteaccount(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
		LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
		if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false{
			http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
		}
		DeleteUser(LOGGEDUSERNAME.Value)
		http.Redirect(w, r, "/signout", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
}

func usersearch(w http.ResponseWriter, r *http.Request) {
	var searchres, topmenu, searchbar, fbutton, tweetpage string // extra search result content
	tweetpage = "<br/><br/><br/>"
	if r.Method == http.MethodPost{
		LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")	// try to retrieve cookie
		LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
		// if user is not logged in, no need to display the top menu. it's fine.
		if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false{
			topmenu = ""
		} else {
			topmenu += "<div class=\"menu\" style= \"text-align:right; list-style: none;\">" +
					"<a href = \"mainpage\"> Home </a>|" +
					"<a href = \"request_info_change\"> Change Profile </a>|" +
					"<a href = \"followinglist\"> Following </a>|" +
					"<a href = \"followerlist\"> Followers </a>|" +
					"<a href = \"signout\"> Log Out</a></div><br/>"
			
			if LOGGEDUSERNAME.Value != r.PostFormValue("targetuser") {
				if afollowsb(LOGGEDUSERNAME.Value, r.PostFormValue("targetuser")) {
					fbutton = MakeFollowButton(r.PostFormValue("targetuser"), false)	// unfollow button
				} else {
					fbutton = MakeFollowButton(r.PostFormValue("targetuser"), true)		// follow button
				}
			}
		}
		if UserExists(r.PostFormValue("targetuser")){
			dispname, profile := GetUserPublicInfo(r.PostFormValue("targetuser"))
			searchres = "<div style=\"text-decoration: underline;\" >" + dispname + "</div><div class=\"userhandle\" style=\"font-size: 10px\">@" + r.PostFormValue("targetuser") + "</div>" + fbutton + "Profile: " + profile + "<br/>"
		}
		posts := GetTweetsBy(r.PostFormValue("targetuser"))
		for _, post := range posts{
			disp_name, _ := GetUserPublicInfo(post.author)
			tweetpage += "<div style=\"text-decoration: underline;\" >" + disp_name + "</div><div class=\"userhandle\" style=\"font-size: 10px\">@" + post.author + "</div>" + "<br/>"
        	tweetpage += "<div class=\"msg\" style=\"padding-left: 20px; font-size: 12px;\">" + post.message + "</div><br/>"
        }
	}
	searchbar = "<form action=\"usersearch\" method = \"post\">" + 
		"Find User <input type = \"text\" name = \"targetuser\" maxlength=\"20\"></input><input type=\"submit\" value=\"Search User\"><br/><br/></form>"
	page, err := ioutil.ReadFile("./basicsiteheader.html")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(w, string(page) + topmenu + searchbar + searchres + tweetpage)
	// if the search results are empty, it just means that the user searched does not exist
}

func followuser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")
	LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false {
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	Follow(LOGGEDUSERNAME.Value, r.PostFormValue("user"))
	http.Redirect(w, r, "/mainpage", http.StatusSeeOther)
}

func followinglist(w http.ResponseWriter, r *http.Request) {
	var myfollowing, menu string;
	LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")
	LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false {
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	menu = "<div class=\"menu\" style= \"text-align:right; list-style: none;\">" +
		"<a href = \"mainpage\"> Home </a>|" +
		"<a href = \"request_info_change\"> Change Profile </a>|" +
		"<a href = \"followinglist\"> Following </a>|" +
		"<a href = \"followerlist\"> Followers </a>|" +
		"<a href = \"signout\"> Log Out</a></div><br/>"
	switch r.Method {
	case http.MethodGet:
		page, err := ioutil.ReadFile("./basicsiteheader.html")
		if err != nil {
			log.Fatal(err)
		}
		flist := GetFollowing(LOGGEDUSERNAME.Value)
		myfollowing += "<ol>"
		for _, followeduser := range flist {
			userprofile := MakeProfileLink(followeduser)
			fbutton := MakeFollowButton (followeduser, false)
			myfollowing += "<li>" + userprofile + "</li><div style = \"display: inline;\">" + fbutton + "</div>"
		}
		myfollowing += "</ol>"
		fmt.Fprintf(w, string(page) + menu + myfollowing)
	case http.MethodPost:
		r.ParseForm()
	}
}

func followerlist(w http.ResponseWriter, r *http.Request) {
	var myfollowers, menu string;
	LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")
	LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false {
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	menu = "<div class=\"menu\" style= \"text-align:right; list-style: none;\">" +
		"<a href = \"mainpage\"> Home </a>|" +
		"<a href = \"request_info_change\"> Change Profile </a>|" +
		"<a href = \"followinglist\"> Following </a>|" +
		"<a href = \"followerlist\"> Followers </a>|" +
		"<a href = \"signout\"> Log Out</a></div><br/>"
	page, err := ioutil.ReadFile("./basicsiteheader.html")
	if err != nil {
		log.Fatal(err)
	}
	flist := GetFollowers(LOGGEDUSERNAME.Value)
	myfollowers += "<ol>"
	for _, follower := range flist {
		userprofile := MakeProfileLink(follower)
		myfollowers += "<li>" + userprofile + "</li>"
	}
	myfollowers += "</ol>"
	fmt.Fprintf(w, string(page) + menu + myfollowers)
}

func unfollowuser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	LOGGEDUSERNAME, err1 := r.Cookie("LOGGEDUSERNAME")
	LOGGEDPASS, err2 := r.Cookie("LOGGEDPASS")
	if err1 != nil || err2 != nil || Authenticate(LOGGEDUSERNAME.Value, LOGGEDPASS.Value) == false {
		http.Redirect(w, r, "/twitterclone", http.StatusSeeOther)
	}
	Unfollow(LOGGEDUSERNAME.Value, r.PostFormValue("user"))
	http.Redirect(w, r, "/mainpage", http.StatusSeeOther)
}

/*****************************************************/

func main() {
	// RequestTablefromApp("GetUserTable")
	http.HandleFunc("/signup", signup)
    http.HandleFunc("/twitterclone", login)
    http.HandleFunc("/AccountCreationFailed", AccountCreationFailed)
    http.HandleFunc("/mainpage", homescreen)
    http.HandleFunc("/signout", signout)
    http.HandleFunc("/tweet", tweet)
    http.HandleFunc("/request_info_change", changeprofile)
    http.HandleFunc("/deleteaccount", deleteaccount)
    http.HandleFunc("/usersearch", usersearch)
    http.HandleFunc("/followuser", followuser)
    http.HandleFunc("/unfollowuser", unfollowuser)
    http.HandleFunc("/followinglist", followinglist)
    http.HandleFunc("/followerlist", followerlist)
    http.ListenAndServe(":8080", nil)
}
