package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/projecthorizon/horizon/internal/models"
)

// JWT secret - in production, use environment variable
var jwtSecret = []byte("projecthorizon-secret-key-change-in-production")

// Claims for JWT
type Claims struct {
	UserID     string            `json:"userId"`
	Username   string            `json:"username"`
	Role       models.Role       `json:"role"`
	Permission models.Permission `json:"permission"`
	jwt.RegisteredClaims
}

func RegisterAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/setup", setupAdmin)
		auth.POST("/login", login)
		auth.GET("/check", checkAuth)
		auth.POST("/logout", logout)
	}

	// Protected user management routes
	users := rg.Group("/users")
	users.Use(AuthMiddleware(), AdminOnly())
	{
		users.GET("", listUsers)
		users.POST("", createUser)
		users.PUT("/:id", updateUser)
		users.DELETE("/:id", deleteUser)
	}
}

// SetupRequest for initial admin setup
type SetupRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

// setupAdmin creates the first admin user
func setupAdmin(c *gin.Context) {
	store := models.GetUserStore()

	// Check if setup already done
	if store.HasUsers() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup already completed"})
		return
	}

	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := store.CreateUser(req.Username, req.Password, models.RoleAdmin, models.PermReadWrite)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto-login after setup
	token, err := generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin account created successfully",
		"user":    user.ToResponse(),
		"token":   token,
	})
}

// LoginRequest for user login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// login authenticates user and returns JWT
func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	store := models.GetUserStore()
	user, err := store.Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user.ToResponse(),
		"token": token,
	})
}

// checkAuth verifies if setup is needed or user is authenticated
func checkAuth(c *gin.Context) {
	store := models.GetUserStore()

	// Check if first-time setup needed
	if !store.HasUsers() {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"setupRequired": true,
		})
		return
	}

	// Check for token
	tokenString := extractToken(c)
	if tokenString == "" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"setupRequired": false,
		})
		return
	}

	claims, err := validateToken(tokenString)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"setupRequired": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"setupRequired": false,
		"user": gin.H{
			"id":         claims.UserID,
			"username":   claims.Username,
			"role":       claims.Role,
			"permission": claims.Permission,
		},
	})
}

// logout - client should delete token
func logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// CreateUserRequest for creating new users
type CreateUserRequest struct {
	Username   string            `json:"username" binding:"required,min=3"`
	Password   string            `json:"password" binding:"required,min=6"`
	Role       models.Role       `json:"role" binding:"required"`
	Permission models.Permission `json:"permission" binding:"required"`
}

func listUsers(c *gin.Context) {
	store := models.GetUserStore()
	users := store.ListUsers()
	c.JSON(http.StatusOK, users)
}

func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store := models.GetUserStore()
	user, err := store.CreateUser(req.Username, req.Password, req.Role, req.Permission)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user.ToResponse())
}

// UpdateUserRequest for updating user
type UpdateUserRequest struct {
	Role       models.Role       `json:"role" binding:"required"`
	Permission models.Permission `json:"permission" binding:"required"`
}

func updateUser(c *gin.Context) {
	id := c.Param("id")

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store := models.GetUserStore()
	user, err := store.UpdateUser(id, req.Role, req.Permission)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")

	// Get current user from context
	currentUser, exists := c.Get("user")
	if exists {
		claims := currentUser.(*Claims)
		if claims.UserID == id {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
			return
		}
	}

	store := models.GetUserStore()

	// Prevent deleting last admin
	user, err := store.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.Role == models.RoleAdmin && store.CountAdmins() <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the last admin"})
		return
	}

	if err := store.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// generateToken creates a JWT for user
func generateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
		Permission: user.Permission,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "projecthorizon",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// validateToken validates JWT and returns claims
func validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}

// extractToken gets token from header or cookie
func extractToken(c *gin.Context) string {
	// Check Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Check cookie
	token, _ := c.Cookie("token")
	return token
}

// AuthMiddleware validates JWT token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		store := models.GetUserStore()

		// Skip auth if no users (first-time setup)
		if !store.HasUsers() {
			c.Next()
			return
		}

		tokenString := extractToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Store claims in context
		c.Set("user", claims)
		c.Next()
	}
}

// AdminOnly middleware restricts to admin users
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		claims := user.(*Claims)
		if claims.Role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		c.Next()
	}
}

// ReadWriteOnly middleware restricts to read-write permission
func ReadWriteOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		claims := user.(*Claims)
		if claims.Permission != models.PermReadWrite {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Write permission required"})
			return
		}

		c.Next()
	}
}
