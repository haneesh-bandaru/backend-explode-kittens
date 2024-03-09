package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var (
	redisClient *redis.Client
)

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file:", err)
	}

	// Initialize Redis client
	uri := os.Getenv("REDIS_URI")
	if uri == "" {
		log.Fatal("REDIS_URI is not set in environment variables")
	}
	opts, err := redis.ParseURL(uri)
	if err != nil {
		log.Fatal("Error parsing Redis URI:", err)
	}
	redisClient = redis.NewClient(opts)
}

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Println("Error decoding request body:", err)
		return
	}

	// Check if the user already exists
	ctx := context.Background()
	val, err := redisClient.Get(ctx, user.Email).Result()
	if err != nil && err != redis.Nil {
		http.Error(w, "Error checking user existence", http.StatusInternalServerError)
		log.Println("Error checking user existence:", err)
		return
	}
	if val != "" {
		// User already exists, return appropriate error response
		errorResponse := ErrorResponse{Message: "User already exists"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Store user data in Redis
	err = redisClient.Set(ctx, user.Email, user.Password, 0).Err()
	if err != nil {
		http.Error(w, "Error storing user data", http.StatusInternalServerError)
		log.Println("Error storing user data:", err)
		return
	}

	// Respond with success message
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "User signed up successfully")
}

func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Get all keys from Redis
	ctx := context.Background()
	keys, err := redisClient.Keys(ctx, "*").Result()
	if err != nil {
		http.Error(w, "Error retrieving user data", http.StatusInternalServerError)
		log.Println("Error retrieving user keys from Redis:", err) // Log the error
		return
	}

	// Iterate over keys and retrieve corresponding values
	var users []User
	for _, key := range keys {
		password, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			log.Printf("Error retrieving user data for email %s: %v", key, err)
			continue
		}

		// Construct user object
		user := User{
			Email:    key,
			Password: password,
		}
		users = append(users, user)
	}

	// Encode user objects to JSON
	jsonData, err := json.Marshal(users)
	if err != nil {
		http.Error(w, "Error encoding user data to JSON", http.StatusInternalServerError)
		log.Println("Error encoding user data to JSON:", err)
		return
	}

	// Set response headers and write JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Println("Error decoding request body:", err)
		return
	}

	// Check if the user exists
	ctx := context.Background()
	password, err := redisClient.Get(ctx, user.Email).Result()
	if err != nil {
		if err == redis.Nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving user data", http.StatusInternalServerError)
		log.Println("Error retrieving user data:", err)
		return
	}

	// Check if the provided password matches the stored password
	if user.Password != password {
		errorResponse := ErrorResponse{Message: "Incorrect password"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Respond with success message
	loginResponse := ErrorResponse{Message: "Login successful"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResponse)
}

func main() {
	// Initialize router
	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/users/signup", SignupHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/users", GetAllUsersHandler).Methods("GET")
	r.HandleFunc("/users/login", LoginHandler).Methods("POST", "OPTIONS")

	// Start server
	fmt.Println("Server is running on port 8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
