package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role represents user access level
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// Permission represents access permissions
type Permission string

const (
	PermReadWrite Permission = "read-write"
	PermReadOnly  Permission = "read-only"
)

// User represents a system user
type User struct {
	ID         string     `json:"id"`
	Username   string     `json:"username"`
	Password   string     `json:"password,omitempty"` // Hashed, omit in JSON responses
	Role       Role       `json:"role"`
	Permission Permission `json:"permission"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// UserResponse is the safe user response without password
type UserResponse struct {
	ID         string     `json:"id"`
	Username   string     `json:"username"`
	Role       Role       `json:"role"`
	Permission Permission `json:"permission"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// ToResponse converts User to safe response
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:         u.ID,
		Username:   u.Username,
		Role:       u.Role,
		Permission: u.Permission,
		CreatedAt:  u.CreatedAt,
	}
}

// UserStore manages user persistence
type UserStore struct {
	mu       sync.RWMutex
	users    map[string]*User
	filePath string
}

var (
	userStore *UserStore
	once      sync.Once
)

// GetUserStore returns singleton UserStore instance
func GetUserStore() *UserStore {
	once.Do(func() {
		userStore = &UserStore{
			users: make(map[string]*User),
		}
		// Try to load from system config, fallback to local
		paths := []string{
			"/etc/projecthorizon/users.json",
			"config/users.json",
		}
		for _, p := range paths {
			if _, err := os.Stat(filepath.Dir(p)); err == nil {
				userStore.filePath = p
				break
			}
		}
		if userStore.filePath == "" {
			userStore.filePath = "config/users.json"
		}
		userStore.load()
	})
	return userStore
}

// GenerateID creates a random ID
func GenerateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares password with hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// load reads users from file
func (s *UserStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No users yet, that's fine
		}
		return err
	}

	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}

	for _, u := range users {
		s.users[u.Username] = u
	}
	return nil
}

// save writes users to file
func (s *UserStore) save() error {
	users := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// CreateUser creates a new user
func (s *UserStore) CreateUser(username, password string, role Role, perm Permission) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user exists
	if _, exists := s.users[username]; exists {
		return nil, errors.New("user already exists")
	}

	// Validate
	if len(username) < 3 {
		return nil, errors.New("username must be at least 3 characters")
	}
	if len(password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:         GenerateID(),
		Username:   username,
		Password:   hashedPassword,
		Role:       role,
		Permission: perm,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	s.users[username] = user

	if err := s.save(); err != nil {
		delete(s.users, username)
		return nil, err
	}

	return user, nil
}

// GetUser retrieves user by username
func (s *UserStore) GetUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// GetUserByID retrieves user by ID
func (s *UserStore) GetUserByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

// Authenticate validates credentials and returns user
func (s *UserStore) Authenticate(username, password string) (*User, error) {
	user, err := s.GetUser(username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !CheckPassword(password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

// ListUsers returns all users
func (s *UserStore) ListUsers() []UserResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]UserResponse, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u.ToResponse())
	}
	return users
}

// UpdateUser updates user properties
func (s *UserStore) UpdateUser(id string, role Role, perm Permission) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.ID == id {
			u.Role = role
			u.Permission = perm
			u.UpdatedAt = time.Now()
			if err := s.save(); err != nil {
				return nil, err
			}
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

// DeleteUser removes a user
func (s *UserStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for username, u := range s.users {
		if u.ID == id {
			delete(s.users, username)
			return s.save()
		}
	}
	return errors.New("user not found")
}

// HasUsers checks if any users exist
func (s *UserStore) HasUsers() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users) > 0
}

// CountAdmins returns number of admin users
func (s *UserStore) CountAdmins() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, u := range s.users {
		if u.Role == RoleAdmin {
			count++
		}
	}
	return count
}
