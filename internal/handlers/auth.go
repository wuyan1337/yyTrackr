package handlers

import (
	"log"
	"net/http"
	"strings"
	"subtrackr/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userService     *service.UserService
	settingsService *service.SettingsService
	sessionService  *service.SessionService
}

func NewAuthHandler(userService *service.UserService, settingsService *service.SettingsService, sessionService *service.SessionService) *AuthHandler {
	return &AuthHandler{
		userService:     userService,
		settingsService: settingsService,
		sessionService:  sessionService,
	}
}

func isValidRedirect(redirect string) bool {
	if len(redirect) > 2048 {
		return false
	}
	return strings.HasPrefix(redirect, "/") && !strings.HasPrefix(redirect, "//")
}

func renderAuthError(c *gin.Context, message string) {
	c.HTML(http.StatusOK, "login-error.html", gin.H{
		"Error": message,
	})
}

func (h *AuthHandler) ShowRegisterPage(c *gin.Context) {
	if h.sessionService.IsAuthenticated(c.Request) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.HTML(http.StatusOK, "register.html", gin.H{
		"Error": c.Query("error"),
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")
	rememberMe := c.PostForm("remember_me") == "on"

	if password != confirmPassword {
		renderAuthError(c, "两次输入的密码不一致")
		return
	}

	user, err := h.userService.Create(username, password)
	if err != nil {
		renderAuthError(c, err.Error())
		return
	}

	if err := h.bootstrapNewUser(user.ID); err != nil {
		log.Printf("bootstrap new user failed for user %d: %v", user.ID, err)
		renderAuthError(c, "初始化账户失败")
		return
	}

	if err := h.sessionService.CreateSession(c.Writer, c.Request, user.ID, rememberMe); err != nil {
		log.Printf("create session failed after register for user %d: %v", user.ID, err)
		renderAuthError(c, "创建登录会话失败")
		return
	}

	c.Header("HX-Redirect", "/")
	c.Status(http.StatusOK)
}

func (h *AuthHandler) bootstrapNewUser(userID uint) error {
	if err := h.userService.AssignOrphanedRecords(userID); err != nil {
		return err
	}

	settings := h.settingsService.ForUser(userID)
	if err := settings.SetCurrency("USD"); err != nil {
		return err
	}
	if err := settings.SetDateFormat("MM/DD/YYYY"); err != nil {
		return err
	}
	if err := settings.SetTheme("gal-violet"); err != nil {
		return err
	}

	return settings.SaveUIPersonalizationConfig(&service.DefaultUIPersonalizationConfig)
}

func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	if !h.userService.HasUsers() {
		c.Redirect(http.StatusFound, "/register")
		return
	}

	if h.sessionService.IsAuthenticated(c.Request) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	redirect := c.Query("redirect")
	if redirect == "" || !isValidRedirect(redirect) {
		redirect = "/"
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"Redirect": redirect,
		"Error":    c.Query("error"),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	if !h.userService.HasUsers() {
		renderAuthError(c, "请先注册账户")
		return
	}

	identifier := c.PostForm("username")
	password := c.PostForm("password")
	rememberMe := c.PostForm("remember_me") == "on"
	redirect := c.PostForm("redirect")

	if redirect == "" || !isValidRedirect(redirect) {
		redirect = "/"
	}

	user, err := h.userService.Authenticate(identifier, password)
	if err != nil {
		renderAuthError(c, err.Error())
		return
	}

	if err := h.sessionService.CreateSession(c.Writer, c.Request, user.ID, rememberMe); err != nil {
		log.Printf("create session failed for login user %d: %v", user.ID, err)
		renderAuthError(c, "创建登录会话失败")
		return
	}

	c.Header("HX-Redirect", redirect)
	c.Status(http.StatusOK)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.sessionService.DestroySession(c.Writer, c.Request); err != nil {
		log.Printf("destroy session failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	if h.userService.HasUsers() {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.Redirect(http.StatusFound, "/register")
}
