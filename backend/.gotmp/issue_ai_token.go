package main

import (
  "fmt"
  "time"
  "k8s-platform-backend/internal/auth"
)

func main() {
  mgr := auth.NewManager("123456789qwertyuiopQWERTYUIOP")
  token, err := mgr.IssueToken(auth.Claims{UserID: 1, Username: "admin"}, 2*time.Hour)
  if err != nil {
    panic(err)
  }
  fmt.Print(token)
}
