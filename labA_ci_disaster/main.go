package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/yaml.v2"
)

type Config struct {
	JWTSecret  string `yaml:"jwt_secret"`
	DBPassword string `yaml:"db_password"`
}

var cfg Config

func loadConfig() {
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("cannot read config: %v", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("cannot parse config: %v", err)
	}
}

func main() {
	loadConfig()

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/secret", secretHandler)

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	log.Println("Listening on :8080")
	log.Fatal(srv.ListenAndServe())
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Простая аутентификация: любой пользователь → выдача JWT
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user": "attacker",
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
	})
	// Здесь используется SigningMethodNone — уязвимо!
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	fmt.Fprintln(w, tokenString)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "No token", http.StatusUnauthorized)
		return
	}
	tokenString := auth[len("Bearer "):]
	// УЯЗВИМАЯ ЧАСТЬ: используем ParseUnverified
	parser := new(jwt.Parser)
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		http.Error(w, "Token parse error", http.StatusForbidden)
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	fmt.Fprintf(w, "Welcome to admin panel, user: %v\n", claims["user"])
}

func secretHandler(w http.ResponseWriter, r *http.Request) {
	// Утечка пароля БД
	fmt.Fprintf(w, "DB_PASSWORD=%s\n", cfg.DBPassword)
}
