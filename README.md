# Distributed-Twitter-Clone
an implementation of social media features (functionality-wise) in GoLang across distributed infrastructure

# Installation
Download or clone the repository
```
git clone https://github.com/Lily-Ng/Distributed-Twitter-Clone.git
```

# How to Run
1) Download the latest Go distribution from https://golang.org/
2) Build from webserver.go, appserver1.go, appserver2.go,and appserver3.go
Ex.
```
go run webserver.go
```
3) Run the resulting webserver executable, then the app servers starting with the first one in order.
4) in browser, navigate to http://localhost:8080/twitterclone

# Functionalities
1) Users may create accounts and log in
2) Users may send text-based tweets
3) Users may follow other users
4) The application server(s) can handle requests from any webserver through a defined communication protocol

# Assumptions
There are multiple geographically distributed application servers, represented by appserver1.go, appserver2.go, and appserver3.go
